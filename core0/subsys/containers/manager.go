package containers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"sync"
	"time"

	base "github.com/g8os/core0/base"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/g8os/core0/base/utils"
	"github.com/garyburd/redigo/redis"
	"github.com/op/go-logging"
	"github.com/pborman/uuid"
)

const (
	cmdContainerCreate    = "corex.create"
	cmdContainerList      = "corex.list"
	cmdContainerDispatch  = "corex.dispatch"
	cmdContainerTerminate = "corex.terminate"
	cmdContainerFind      = "corex.find"

	coreXResponseQueue = "corex:results"
	coreXBinaryName    = "coreX"

	redisSocketSrc    = "/var/run/redis.socket"
	DefaultBridgeName = "core0"
)

var (
	BridgeIP          = []byte{172, 18, 0, 1}
	DefaultBridgeIP   = fmt.Sprintf("%d.%d.%d.%d", BridgeIP[0], BridgeIP[1], BridgeIP[2], BridgeIP[3])
	DefaultBridgeCIDR = fmt.Sprintf("%s/16", DefaultBridgeIP)
)

var (
	log = logging.MustGetLogger("containers")
)

type NetworkConfig struct {
	Dhcp    bool     `json:"dhcp"`
	CIDR    string   `json:"cidr"`
	Gateway string   `json:"gateway"`
	DNS     []string `json:"dns"`
}

type Nic struct {
	Type      string        `json:"type"`
	ID        string        `json:"id"`
	HWAddress string        `json:"hwaddr"`
	Config    NetworkConfig `json:"config"`
}

type ContainerCreateArguments struct {
	Root        string            `json:"root"`         //Root plist
	Mount       map[string]string `json:"mount"`        //data disk mounts.
	HostNetwork bool              `json:"host_network"` //share host networking stack
	Nics        []Nic             `json:"nics"`         //network setup (only respected if HostNetwork is false)
	Port        map[int]int       `json:"port"`         //port forwards (only if default networking is enabled)
	Hostname    string            `json:"hostname"`     //hostname
	Storage     string            `json:"storage"`      //ardb storage needed for g8ufs mounts.
	Tags        []string          `json:"tags"`         //for searching containers
}

type ContainerDispatchArguments struct {
	Container uint16       `json:"container"`
	Command   core.Command `json:"command"`
}

func (c *ContainerCreateArguments) Validate() error {
	if c.Root == "" {
		return fmt.Errorf("root plist is required")
	}

	for host, guest := range c.Mount {
		u, err := url.Parse(host)
		if err != nil {
			return fmt.Errorf("invalid host mount: %s", err)
		}
		if u.Scheme != "" {
			//probably a plist
			continue
		}
		p := u.Path
		if !path.IsAbs(p) {
			return fmt.Errorf("host path '%s' must be absolute", host)
		}
		if !path.IsAbs(guest) {
			return fmt.Errorf("guest path '%s' must be absolute", guest)
		}
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return fmt.Errorf("host path '%s' does not exist", p)
		}
	}

	for host, guest := range c.Port {
		if host < 0 || host > 65535 {
			return fmt.Errorf("invalid host port '%d'", host)
		}
		if guest < 0 || guest > 65535 {
			return fmt.Errorf("invalid guest port '%d'", guest)
		}
	}

	//validating networking
	var def int
	for _, net := range c.Nics {
		switch net.Type {
		case "default":
			def++
		case "vlan":
		case "vxlan":
		case "zerotier":
		default:
			return fmt.Errorf("unsupported network type '%s'", net.Type)
		}
	}

	if def > 1 {
		return fmt.Errorf("only one default network is allowed")
	}

	return nil
}

type containerManager struct {
	sequence uint16
	seqM     sync.Mutex

	containers map[uint16]*container
	conM       sync.RWMutex

	pool *redis.Pool

	internal *internalRouter
	sinks    map[string]base.SinkClient
}

/*
WARNING:
	Code here assumes that redis-server is started by core0 by the configuration files. If it wasn't started or failed
	to start, commands like core.create, core.dispatch, etc... will fail.
TODO:
	May be make redis-server start part of the bootstrap process without the need to depend on external configuration
	to run it.
*/

type Container interface {
	ID() uint16
	Arguments() ContainerCreateArguments
}

type ContainerManager interface {
	Dispatch(id uint16, cmd *core.Command) (*core.JobResult, error)
	GetWithTags(tags ...string) []Container
	GetOneWithTags(tags ...string) Container
	Of(id uint16) Container
}

func ContainerSubsystem(sinks map[string]base.SinkClient) (ContainerManager, error) {
	containerMgr := &containerManager{
		pool:       utils.NewRedisPool("unix", redisSocketSrc, ""),
		containers: make(map[uint16]*container),
		sinks:      sinks,
		internal:   newInternalRouter(),
	}

	pm.CmdMap[cmdContainerCreate] = process.NewInternalProcessFactory(containerMgr.create)
	pm.CmdMap[cmdContainerList] = process.NewInternalProcessFactory(containerMgr.list)
	pm.CmdMap[cmdContainerDispatch] = process.NewInternalProcessFactory(containerMgr.dispatch)
	pm.CmdMap[cmdContainerTerminate] = process.NewInternalProcessFactory(containerMgr.terminate)
	pm.CmdMap[cmdContainerFind] = process.NewInternalProcessFactory(containerMgr.find)

	if err := containerMgr.setUpDefaultBridge(); err != nil {
		return nil, err
	}

	go containerMgr.startForwarder()

	return containerMgr, nil
}

func (m *containerManager) setUpDefaultBridge() error {
	cmd := &core.Command{
		ID:      uuid.New(),
		Command: "bridge.create",
		Arguments: core.MustArguments(
			core.M{
				"name": DefaultBridgeName,
				"network": core.M{
					"nat":  true,
					"mode": "static",
					"settings": core.M{
						"cidr": DefaultBridgeCIDR,
					},
				},
			},
		),
	}

	runner, err := pm.GetManager().RunCmd(cmd)
	if err != nil {
		return err
	}
	result := runner.Wait()
	if result.State != core.StateSuccess {
		return fmt.Errorf("failed to create default container bridge: %s", result.Data)
	}

	return nil
}

func (m *containerManager) forwardNext() error {
	db := m.pool.Get()
	defer db.Close()

	payload, err := redis.ByteSlices(db.Do("BLPOP", coreXResponseQueue, 0))
	if err != nil {
		return err
	}

	var result core.JobResult
	if err := json.Unmarshal(payload[1], &result); err != nil {
		log.Errorf("Failed to load command: %s", err)
		return nil //no wait.
	}

	//use command tags for routing.
	if result.Tags == string(InternalRoute) {
		m.internal.Route(&result)
	} else if sink, ok := m.sinks[result.Tags]; ok {
		log.Debugf("Forwarding job result to %s", result.Tags)
		return sink.Respond(&result)
	} else {
		log.Warningf("Received a corex result for an unknown sink: %s", result.Tags)
	}

	return nil
}

func (m *containerManager) startForwarder() {
	log.Debugf("Start container results forwarder")
	for {
		if err := m.forwardNext(); err != nil {
			log.Warningf("Failed to forward command result: %s", err)
			time.Sleep(2 * time.Second)
		}
	}
}

func (m *containerManager) getNextSequence() uint16 {
	m.seqM.Lock()
	defer m.seqM.Unlock()
	m.sequence += 1
	return m.sequence
}

func (m *containerManager) create(cmd *core.Command) (interface{}, error) {
	var args ContainerCreateArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	if err := args.Validate(); err != nil {
		return nil, err
	}

	id := m.getNextSequence()
	c := newContainer(m, id, cmd.Route, args)

	m.conM.Lock()
	m.containers[id] = c
	m.conM.Unlock()

	if err := c.Start(); err != nil {
		return nil, err
	}

	return id, nil
}

//cleanup is called when a container terminates.
func (m *containerManager) cleanup(id uint16) {
	m.conM.Lock()
	defer m.conM.Unlock()
	delete(m.containers, id)
}

type ContainerInfo struct {
	process.ProcessStats
	Container Container `json:"container"`
}

func (m *containerManager) list(cmd *core.Command) (interface{}, error) {
	containers := make(map[uint16]ContainerInfo)

	m.conM.RLock()
	defer m.conM.RUnlock()
	for id, c := range m.containers {
		name := fmt.Sprintf("core-%d", id)
		runner, ok := pm.GetManager().Runners()[name]
		if !ok {
			continue
		}
		ps := runner.Process()
		var state process.ProcessStats
		if ps != nil {
			if stater, ok := ps.(process.Stater); ok {
				state = *(stater.Stats())
			}
		}
		containers[id] = ContainerInfo{
			ProcessStats: state,
			Container:    c,
		}
	}

	return containers, nil
}

func (m *containerManager) getCoreXQueue(id uint16) string {
	return fmt.Sprintf("core:%v", id)
}

func (m *containerManager) pushToContainer(container uint16, cmd *core.Command) error {
	db := m.pool.Get()
	defer db.Close()

	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	_, err = db.Do("RPUSH", m.getCoreXQueue(container), string(data))

	return err
}

func (m *containerManager) dispatch(cmd *core.Command) (interface{}, error) {
	var args ContainerDispatchArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	if args.Container <= 0 {
		return nil, fmt.Errorf("invalid container id")
	}

	m.conM.RLock()
	_, ok := m.containers[args.Container]
	m.conM.RUnlock()

	if !ok {
		return nil, fmt.Errorf("container does not exist")
	}

	id := uuid.New()
	args.Command.ID = id
	args.Command.Tags = string(cmd.Route)

	if err := m.pushToContainer(args.Container, &args.Command); err != nil {
		return nil, err
	}

	return id, nil
}

//Dispatch command to container with ID (id)
func (m *containerManager) Dispatch(id uint16, cmd *core.Command) (*core.JobResult, error) {
	cmd.ID = uuid.New()
	cmd.Tags = string(InternalRoute)

	m.internal.Prepare(cmd.ID)
	if err := m.pushToContainer(id, cmd); err != nil {
		return nil, err
	}
	job := m.internal.Get(cmd.ID)
	if job == nil {
		return nil, fmt.Errorf("timeout")
	}

	return job, nil
}

type ContainerTerminateArguments struct {
	Container uint16 `json:"container"`
}

func (m *containerManager) terminate(cmd *core.Command) (interface{}, error) {
	var args ContainerTerminateArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	coreID := fmt.Sprintf("core-%d", args.Container)
	return nil, pm.GetManager().Kill(coreID)
}

type ContainerFindArguments struct {
	Tags []string `json:"tags"`
}

func (m *containerManager) find(cmd *core.Command) (interface{}, error) {
	var args ContainerFindArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	containers := m.GetWithTags(args.Tags...)
	result := make(map[uint16]ContainerInfo)
	for _, c := range containers {
		name := fmt.Sprintf("core-%d", c.ID())
		runner, ok := pm.GetManager().Runners()[name]
		if !ok {
			continue
		}
		ps := runner.Process()
		var state process.ProcessStats
		if ps != nil {
			if stater, ok := ps.(process.Stater); ok {
				state = *(stater.Stats())
			}
		}

		result[c.ID()] = ContainerInfo{
			ProcessStats: state,
			Container:    c,
		}
	}

	return result, nil
}

func (m *containerManager) GetWithTags(tags ...string) []Container {
	m.conM.RLock()
	defer m.conM.RUnlock()

	var result []Container
loop:
	for _, c := range m.containers {
		for _, tag := range tags {
			if !utils.InString(c.Args.Tags, tag) {
				continue loop
			}
		}
		result = append(result, c)
	}

	return result
}

func (m *containerManager) GetOneWithTags(tags ...string) Container {
	result := m.GetWithTags(tags...)
	if len(result) > 0 {
		return result[0]
	}

	return nil
}

func (m *containerManager) Of(id uint16) Container {
	m.conM.RLock()
	defer m.conM.RUnlock()
	cont, _ := m.containers[id]
	return cont
}
