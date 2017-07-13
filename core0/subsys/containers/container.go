package containers

import (
	"fmt"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"os"
	"path"
	"sync"
	"syscall"
	"time"
)

const (
	OVSTag       = "ovs"
	OVSBackPlane = "backplane"
	OVSVXBackend = "vxbackend"
)

var (
	devicesToBind = []string{"random", "urandom", "null"}
)

type container struct {
	id     uint16
	runner pm.Runner
	mgr    *containerManager
	route  core.Route
	Args   ContainerCreateArguments `json:"arguments"`
	Root   string                   `json:"root"`
	PID    int                      `json:"pid"`

	zt    pm.Runner
	zterr error
	zto   sync.Once

	channel     process.Channel
	forwardChan chan *core.Command
}

func newContainer(mgr *containerManager, id uint16, route core.Route, args ContainerCreateArguments) *container {
	c := &container{
		mgr:         mgr,
		id:          id,
		route:       route,
		Args:        args,
		forwardChan: make(chan *core.Command),
	}
	c.Root = c.root()
	return c
}

func (c *container) ID() uint16 {
	return c.id
}

func (c *container) dispatch(cmd *core.Command) error {
	select {
	case c.forwardChan <- cmd:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("failed to dispatch command to container, check system logs for errors")
	}
	return nil
}

func (c *container) Arguments() ContainerCreateArguments {
	return c.Args
}

func (c *container) Start() (err error) {
	coreID := fmt.Sprintf("core-%d", c.id)

	defer func() {
		if err != nil {
			c.cleanup()
		}
	}()

	if err = c.sandbox(); err != nil {
		log.Errorf("error in container mount: %s", err)
		return
	}

	if err = c.preStart(); err != nil {
		log.Errorf("error in container prestart: %s", err)
		return
	}

	args := []string{
		"-hostname", c.Args.Hostname,
	}

	if !c.Args.Privileged {
		args = append(args, "-unprivileged")
	}

	mgr := pm.GetManager()
	extCmd := &core.Command{
		ID:    coreID,
		Route: c.route,
		Arguments: core.MustArguments(
			process.ContainerCommandArguments{
				Name:        "/coreX",
				Chroot:      c.root(),
				Dir:         "/",
				HostNetwork: c.Args.HostNetwork,
				Args:        args,
				Env: map[string]string{
					"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
					"HOME": "/",
				},
			},
		),
	}

	onpid := &pm.PIDHook{
		Action: c.onStart,
	}

	onexit := &pm.ExitHook{
		Action: c.onExit,
	}
	var runner pm.Runner
	runner, err = mgr.NewRunner(extCmd, process.NewContainerProcess, onpid, onexit)
	if err != nil {
		log.Errorf("error in container runner: %s", err)
		return
	}

	c.runner = runner
	return
}

func (c *container) Terminate() error {
	if c.runner == nil {
		return fmt.Errorf("container was not started")
	}
	c.runner.Terminate()
	c.runner.Wait()
	return nil
}

func (c *container) preStart() error {
	if c.Args.HostNetwork {
		return c.preStartHostNetworking()
	}

	if err := c.preStartIsolatedNetworking(); err != nil {
		return err
	}

	return nil
}

func (c *container) onStart(pid int) {
	//get channel
	ps := c.runner.Process()
	if ps, ok := ps.(process.ContainerProcess); !ok {
		log.Errorf("not a valid container process")
		c.runner.Terminate()
		return
	} else {
		c.channel = ps.Channel()
	}

	c.PID = pid
	if !c.Args.Privileged {
		c.mgr.cgroup.Task(pid)
	}

	if err := c.postStart(); err != nil {
		log.Errorf("Container post start error: %s", err)
		//TODO. Should we shut the container down?
	}

	go c.rewind()
	go c.forward()
}

func (c *container) onExit(state bool) {
	log.Debugf("Container %v exited with state %v", c.id, state)
	c.cleanup()
}

func (c *container) cleanup() {
	log.Debugf("cleaning up container-%d", c.id)
	defer c.mgr.unsetContainer(c.id)

	close(c.forwardChan)
	if c.channel != nil {
		c.channel.Close()
	}

	c.destroyNetwork()

	if err := c.unMountAll(); err != nil {
		log.Errorf("unmounting container-%d was not clean", err)
	}
}

func (c *container) cleanSandbox() {
	if c.getFSType(BackendBaseDir) == "btrfs" {
		pm.GetManager().System("btrfs", "subvolume", "delete", path.Join(BackendBaseDir, c.name()))
	} else {
		os.RemoveAll(path.Join(BackendBaseDir, c.name()))
	}

	os.RemoveAll(c.root())
}

func (c *container) namespace() error {
	sourceNs := fmt.Sprintf("/proc/%d/ns/net", c.PID)
	os.MkdirAll("/run/netns", 0755)
	targetNs := fmt.Sprintf("/run/netns/%v", c.id)

	if f, err := os.Create(targetNs); err == nil {
		f.Close()
	}

	if err := syscall.Mount(sourceNs, targetNs, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("namespace mount: %s", err)
	}

	return nil
}

func (c *container) postStart() error {
	if c.Args.HostNetwork {
		return nil
	}

	if err := c.postStartIsolatedNetworking(); err != nil {
		log.Errorf("isolated networking error: %s", err)
		return err
	}

	return nil
}
