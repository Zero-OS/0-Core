package containers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/op/go-logging"
	"github.com/pborman/uuid"
	"github.com/zero-os/0-core/apps/core0/helper/socat"
	"github.com/zero-os/0-core/apps/core0/screen"
	"github.com/zero-os/0-core/apps/core0/subsys/cgroups"
	"github.com/zero-os/0-core/apps/core0/transport"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/settings"
	"github.com/zero-os/0-core/base/utils"
)

const (
	cmdContainerCreate            = "corex.create"
	cmdContainerCreateSync        = "corex.create-sync"
	cmdContainerList              = "corex.list"
	cmdContainerDispatch          = "corex.dispatch"
	cmdContainerTerminate         = "corex.terminate"
	cmdContainerFind              = "corex.find"
	cmdContainerZerotierInfo      = "corex.zerotier.info"
	cmdContainerZerotierList      = "corex.zerotier.list"
	cmdContainerNicAdd            = "corex.nic-add"
	cmdContainerNicRemove         = "corex.nic-remove"
	cmdContainerBackup            = "corex.backup"
	cmdContainerRestore           = "corex.restore"
	cmdContainerPortForwardAdd    = "corex.portforward-add"
	cmdContainerPortForwardRemove = "corex.portforward-remove"

	coreXResponseQueue = "corex:results"
	coreXBinaryName    = "coreX"

	redisSocketSrc      = "/var/run/redis.socket"
	DefaultBridgeName   = "core0"
	ContainersHardLimit = 1000
)

const (
	NicStateConfigured = NicState("configured")
	NicStateDestroyed  = NicState("destroyed")
	NicStateUnknown    = NicState("unknown")
	NicStateError      = NicState("error")
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

type NicState string

type Nic struct {
	Type      string        `json:"type"`
	ID        string        `json:"id"`
	HWAddress string        `json:"hwaddr"`
	Name      string        `json:"name,omitempty"`
	Config    NetworkConfig `json:"config"`
	Monitor   bool          `json:"monitor"`

	State NicState `json:"state"`
}

type ContainerCreateArguments struct {
	Root        string            `json:"root"`         //Root plist
	Mount       map[string]string `json:"mount"`        //data disk mounts.
	HostNetwork bool              `json:"host_network"` //share host networking stack
	Identity    string            `json:"identity"`     //zerotier identity
	Nics        []*Nic            `json:"nics"`         //network setup (only respected if HostNetwork is false)
	Port        map[string]int    `json:"port"`         //port forwards (only if default networking is enabled)
	Privileged  bool              `json:"privileged"`   //Apply cgroups and capabilities limitations on the container
	Hostname    string            `json:"hostname"`     //hostname
	Storage     string            `json:"storage"`      //ardb storage needed for g8ufs mounts.
	Name        string            `json:"name"`         //for searching containers
	Tags        pm.Tags           `json:"tags"`         //for searching containers
	Env         map[string]string `json:"env"`          //environment variables.
}

type ContainerDispatchArguments struct {
	Container uint16     `json:"container"`
	Command   pm.Command `json:"command"`
}

type containerPortForward struct {
	Container     uint16 `json:"container"`
	ContainerPort int    `json:"container_port"`
	HostPort      string `json:"host_port"`
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
		if !socat.ValidHost(host) {
			return fmt.Errorf("invalid host port '%s'", host)
		}
		if guest < 0 || guest > 65535 {
			return fmt.Errorf("invalid guest port '%d'", guest)
		}
	}

	//validating networking
	brcounter := make(map[string]int)
	for _, nic := range c.Nics {
		if nic.State == NicStateDestroyed {
			continue
		}
		switch nic.Type {
		case "default":
			brcounter[DefaultBridgeName]++
			if brcounter[DefaultBridgeName] > 1 {
				return fmt.Errorf("only one default network is allowed")
			}
		case "bridge":
			if nic.ID == DefaultBridgeName {
				return fmt.Errorf("cannot use bridge %s with nic type 'bridge', please use type default instead", DefaultBridgeName)
			}
			brcounter[nic.ID]++
			if brcounter[nic.ID] > 1 {
				return fmt.Errorf("connecting to bridge '%s' more than one time is not allowed", nic.ID)
			}
		case "macvlan":
			brcounter[nic.ID]++
			if brcounter[nic.ID] > 1 {
				return fmt.Errorf("connecting to link '%s' more than one time is not allowed", nic.ID)
			}
		case "vlan":
		case "vxlan":
		case "zerotier":
		default:
			return fmt.Errorf("unsupported network type '%s'", nic.Type)
		}
	}

	nameset := make(map[string]byte)
	for _, nic := range c.Nics {
		if nic.State == NicStateDestroyed {
			continue
		}
		if nic.Name != "" {
			if _, ok := nameset[nic.Name]; ok {
				return fmt.Errorf("name '%v' is passed twice in the container", nic.Name)
			} else {
				nameset[nic.Name] = 1
			}
			if len(nic.Name) > 15 { //linux limit on interface name
				return fmt.Errorf("invalid name '%s' too long", nic.Name)
			}
			if nic.Name == "default" { //probably we need to expand this list with more reserved names
				//`default` is not allowed by linux for some reason.
				return fmt.Errorf("invalid name `%s`", nic.Name)
			}
			//avoid conflict with eth or zt
			if strings.HasPrefix(nic.Name, "eth") || strings.HasPrefix(nic.Name, "zt") {
				return fmt.Errorf("name '%v' cannot be used as it is started with eth or zt", nic.Name)
			}
		}
	}

	return nil
}

type containerManager struct {
	sequence uint16
	seqM     sync.Mutex

	containers map[uint16]*container
	conM       sync.RWMutex

	cell   *screen.RowCell
	cgroup cgroups.Group

	sink *transport.Sink
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
	Dispatch(id uint16, cmd *pm.Command) (*pm.JobResult, error)
	GetWithTags(tags ...string) []Container
	GetOneWithTags(tags ...string) Container
	Of(id uint16) Container
}

func ContainerSubsystem(sink *transport.Sink, cell *screen.RowCell) (ContainerManager, error) {
	if err := cgroups.Init(); err != nil {
		return nil, err
	}

	containerMgr := &containerManager{
		containers: make(map[uint16]*container),
		sink:       sink,
		cell:       cell,
	}

	cell.Text = "Containers: 0"

	if err := containerMgr.setUpCGroups(); err != nil {
		return nil, err
	}
	if err := containerMgr.setUpDefaultBridge(); err != nil {
		return nil, err
	}

	pm.RegisterBuiltIn(cmdContainerCreate, containerMgr.create)
	pm.RegisterBuiltIn(cmdContainerCreateSync, containerMgr.createSync)
	pm.RegisterBuiltIn(cmdContainerList, containerMgr.list)
	pm.RegisterBuiltIn(cmdContainerDispatch, containerMgr.dispatch)
	pm.RegisterBuiltIn(cmdContainerTerminate, containerMgr.terminate)
	pm.RegisterBuiltIn(cmdContainerFind, containerMgr.find)
	pm.RegisterBuiltIn(cmdContainerNicAdd, containerMgr.nicAdd)
	pm.RegisterBuiltIn(cmdContainerNicRemove, containerMgr.nicRemove)
	pm.RegisterBuiltIn(cmdContainerPortForwardAdd, containerMgr.portforwardAdd)
	pm.RegisterBuiltIn(cmdContainerPortForwardRemove, containerMgr.portforwardRemove)
	pm.RegisterBuiltIn(cmdContainerBackup, containerMgr.backup)
	pm.RegisterBuiltIn(cmdContainerRestore, containerMgr.restore)

	//container specific info
	pm.RegisterBuiltIn(cmdContainerZerotierInfo, containerMgr.ztInfo)
	pm.RegisterBuiltIn(cmdContainerZerotierList, containerMgr.ztList)

	return containerMgr, nil
}

func (m *containerManager) setUpCGroups() error {
	devices, err := cgroups.GetGroup("corex", cgroups.DevicesSubsystem)
	if err != nil {
		return err
	}

	if devices, ok := devices.(cgroups.DevicesGroup); ok {
		devices.Deny("a")
		for _, spec := range []string{
			"c 1:5 rwm",
			"c 1:3 rwm",
			"c 1:9 rwm",
			"c 1:7 rwm",
			"c 1:8 rwm",
			"c 5:0 rwm",
			"c 5:1 rwm",
			"c 5:2 rwm",
			"c *:* m",
			"c 136:* rwm",
			"c 10:200 rwm",
		} {
			devices.Allow(spec)
		}
	} else {
		return fmt.Errorf("failed to setup devices cgroups")
	}

	m.cgroup = devices
	return nil
}

func (m *containerManager) setUpDefaultBridge() error {
	cmd := &pm.Command{
		ID:      uuid.New(),
		Command: "bridge.create",
		Arguments: pm.MustArguments(
			pm.M{
				"name": DefaultBridgeName,
				"network": pm.M{
					"nat":  true,
					"mode": "static",
					"settings": pm.M{
						"cidr": DefaultBridgeCIDR,
					},
				},
			},
		),
	}

	job, err := pm.Run(cmd)
	if err != nil {
		return err
	}
	result := job.Wait()
	if result.State != pm.StateSuccess {
		return fmt.Errorf("failed to create default container bridge: %s", result.Data)
	}

	return nil
}

func (m *containerManager) getNextSequence() uint16 {
	m.seqM.Lock()
	defer m.seqM.Unlock()
	//get a read lock on the container dict as well
	m.conM.RLock()
	defer m.conM.RUnlock()

	for {
		m.sequence += 1
		if m.sequence != 0 && m.sequence < math.MaxUint16 {
			if _, ok := m.containers[m.sequence]; !ok {
				break
			}
		}
	}
	return m.sequence
}

func (m *containerManager) setContainer(id uint16, c *container) {
	m.conM.Lock()
	defer m.conM.Unlock()
	m.containers[id] = c
	m.cell.Text = fmt.Sprintf("Containers: %d", len(m.containers))
	screen.Refresh()
}

//cleanup is called when a container terminates.
func (m *containerManager) unsetContainer(id uint16) {
	m.conM.Lock()
	defer m.conM.Unlock()
	delete(m.containers, id)
	m.cell.Text = fmt.Sprintf("Containers: %d", len(m.containers))
	screen.Refresh()
}

func (m *containerManager) nicAdd(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Container uint16 `json:"container"`
		Nic       Nic    `json:"nic"`
	}
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	m.conM.RLock()
	defer m.conM.RUnlock()
	container, ok := m.containers[args.Container]
	if !ok {
		return nil, pm.NotFoundError(fmt.Errorf("container does not exist"))
	}

	if container.Args.HostNetwork {
		return nil, pm.BadRequestError(fmt.Errorf("cannot add a nic in host network mode"))
	}

	args.Nic.State = NicStateUnknown

	idx := len(container.Args.Nics)
	container.Args.Nics = append(container.Args.Nics, &args.Nic)

	if err := container.Args.Validate(); err != nil {
		l := container.Args.Nics
		container.Args.Nics = l[:len(l)-1]
		return nil, pm.BadRequestError(err)
	}

	if err := container.preStartNetwork(idx, &args.Nic); err != nil {
		return nil, err
	}

	if err := container.postStartNetwork(idx, &args.Nic); err != nil {
		return nil, err
	}

	return nil, nil
}

func (m *containerManager) nicRemove(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Container uint16 `json:"container"`
		Index     int    `json:"index"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	m.conM.RLock()
	defer m.conM.RUnlock()
	container, ok := m.containers[args.Container]
	if !ok {
		return nil, pm.NotFoundError(fmt.Errorf("container does not exist"))
	}

	if args.Index < 0 || args.Index >= len(container.Args.Nics) {
		return nil, pm.BadRequestError(fmt.Errorf("nic index out of range"))
	}
	nic := container.Args.Nics[args.Index]
	if nic.State != NicStateConfigured {
		return nil, pm.PreconditionFailedError(fmt.Errorf("nic is in '%s' state", nic.State))
	}

	if nic.Type == "zerotier" {
		//special handling for zerotier networks
		if err := container.leaveZerotierNetwork(args.Index, nic.ID); err != nil {
			nic.State = NicStateError
			return nil, err
		}

		nic.State = NicStateDestroyed
		return nil, nil
	}

	if nic.Type == "macvlan" {
		return nil, container.unLink(args.Index, nic)
	}

	var ovs Container
	if nic.Type == "vlan" || nic.Type == "vxlan" {
		ovs = m.GetOneWithTags("ovs")
	}

	return nil, container.unBridge(args.Index, nic, ovs)
}

func (m *containerManager) createContainer(args ContainerCreateArguments) (*container, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	m.conM.RLock()
	count := len(m.containers)
	m.conM.RUnlock()
	limit := settings.Settings.Containers.MaxCount
	if limit == 0 {
		limit = ContainersHardLimit
	}

	if count >= limit {
		return nil, pm.ServiceUnavailableError(fmt.Errorf("reached the hard limit of %d containers", count))
	}

	id := m.getNextSequence()
	c := newContainer(m, id, args)
	m.setContainer(id, c)

	if _, err := c.Start(); err != nil {
		return nil, err
	}

	return c, nil
}

func (m *containerManager) createSync(cmd *pm.Command) (interface{}, error) {
	var args ContainerCreateArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		log.Errorf("invalid container params: %s", err)
		return nil, err
	}

	args.Tags = cmd.Tags
	container, err := m.createContainer(args)
	if err != nil {
		log.Errorf("failed to start container: %s", err)
		return nil, err
	}

	//after waiting we probably need to return the full result!
	return container.runner.Wait(), nil
}

func (m *containerManager) create(cmd *pm.Command) (interface{}, error) {
	var args ContainerCreateArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	args.Tags = cmd.Tags
	container, err := m.createContainer(args)
	if err != nil {
		return nil, err
	}

	return container.id, nil
}

type ContainerInfo struct {
	pm.ProcessStats
	Container Container `json:"container"`
}

func (m *containerManager) list(cmd *pm.Command) (interface{}, error) {
	containers := make(map[uint16]ContainerInfo)

	m.conM.RLock()
	defer m.conM.RUnlock()
	for id, c := range m.containers {
		name := fmt.Sprintf("core-%d", id)
		job, ok := pm.JobOf(name)
		if !ok {
			continue
		}
		ps := job.Process()
		var state pm.ProcessStats
		if ps != nil {
			if stater, ok := ps.(pm.Stater); ok {
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

func (m *containerManager) pushToContainer(container *container, cmd *pm.Command) error {
	m.sink.Flag(cmd.ID)
	return container.dispatch(cmd)
}

func (m *containerManager) dispatch(cmd *pm.Command) (interface{}, error) {
	var args ContainerDispatchArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	if args.Container <= 0 {
		return nil, fmt.Errorf("invalid container id")
	}

	m.conM.RLock()
	cont, ok := m.containers[args.Container]
	m.conM.RUnlock()

	if !ok {
		return nil, fmt.Errorf("container does not exist")
	}

	if args.Command.ID == "" {
		args.Command.ID = uuid.New()
	}

	if err := m.pushToContainer(cont, &args.Command); err != nil {
		return nil, err
	}

	return args.Command.ID, nil
}

//Dispatch command to container with ID (id)
func (m *containerManager) Dispatch(id uint16, cmd *pm.Command) (*pm.JobResult, error) {
	cmd.ID = uuid.New()

	m.conM.RLock()
	cont, ok := m.containers[id]
	m.conM.RUnlock()

	if !ok {
		return nil, fmt.Errorf("container does not exist")
	}

	if err := m.pushToContainer(cont, cmd); err != nil {
		return nil, err
	}

	return m.sink.GetResult(cmd.ID, transport.ReturnExpire)
}

type ContainerArguments struct {
	Container uint16 `json:"container"`
}

func (m *containerManager) terminate(cmd *pm.Command) (interface{}, error) {
	var args ContainerArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	m.conM.RLock()
	container, ok := m.containers[args.Container]
	m.conM.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no container with id '%d'", args.Container)
	}

	return nil, container.Terminate()
}

type ContainerFindArguments struct {
	Tags []string `json:"tags"`
}

func (m *containerManager) find(cmd *pm.Command) (interface{}, error) {
	var args ContainerFindArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	containers := m.GetWithTags(args.Tags...)
	result := make(map[uint16]ContainerInfo)
	for _, c := range containers {
		name := fmt.Sprintf("core-%d", c.ID())
		job, ok := pm.JobOf(name)
		if !ok {
			continue
		}
		ps := job.Process()
		var state pm.ProcessStats
		if ps != nil {
			if stater, ok := ps.(pm.Stater); ok {
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

func (m *containerManager) portforwardAdd(cmd *pm.Command) (interface{}, error) {
	var args containerPortForward
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	m.conM.RLock()
	defer m.conM.RUnlock()

	container, ok := m.containers[args.Container]
	if !ok {
		return nil, pm.NotFoundError(fmt.Errorf("container does not exist"))
	}
	var defaultNic bool
	for _, nic := range container.Args.Nics {
		if nic.Type == "default" {
			defaultNic = true
			break
		}
	}
	if !defaultNic {
		return nil, fmt.Errorf("Container doesn't have a default nic")
	}

	if err := container.setPortForward(args.HostPort, args.ContainerPort); err != nil {
		return nil, err
	}

	container.Args.Port[args.HostPort] = args.ContainerPort
	return nil, nil
}

func (m *containerManager) portforwardRemove(cmd *pm.Command) (interface{}, error) {
	var args containerPortForward
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	m.conM.RLock()
	defer m.conM.RUnlock()

	container, ok := m.containers[args.Container]
	if !ok {
		return nil, pm.NotFoundError(fmt.Errorf("container does not exist"))
	}

	if err := socat.RemovePortForward(container.forwardId(), args.HostPort, args.ContainerPort); err != nil {
		return nil, err
	}
	delete(container.Args.Port, args.HostPort)
	return nil, nil
}
