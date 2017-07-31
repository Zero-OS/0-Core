package containers

import (
	"fmt"
	"github.com/zero-os/0-core/base/pm"
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
	runner pm.Job
	mgr    *containerManager
	Args   ContainerCreateArguments `json:"arguments"`
	Root   string                   `json:"root"`
	PID    int                      `json:"pid"`

	zt    pm.Job
	zterr error
	zto   sync.Once

	channel     pm.Channel
	forwardChan chan *pm.Command
}

func newContainer(mgr *containerManager, id uint16, args ContainerCreateArguments) *container {
	c := &container{
		mgr:         mgr,
		id:          id,
		Args:        args,
		forwardChan: make(chan *pm.Command),
	}
	c.Root = c.root()
	return c
}

func (c *container) ID() uint16 {
	return c.id
}

func (c *container) dispatch(cmd *pm.Command) error {
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

func (c *container) Start() (runner pm.Job, err error) {
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

	//Set a Default Env and merge it with environment map from args
	env := map[string]string{
		"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME": "/",
	}
	for key, value := range c.Args.Env {
		env[key] = value
	}

	extCmd := &pm.Command{
		ID: coreID,
		Arguments: pm.MustArguments(
			pm.ContainerCommandArguments{
				Name:        "/coreX",
				Chroot:      c.root(),
				Dir:         "/",
				HostNetwork: c.Args.HostNetwork,
				Args:        args,
				Env:         env,
			},
		),
	}

	onpid := &pm.PIDHook{
		Action: c.onStart,
	}

	onexit := &pm.ExitHook{
		Action: c.onExit,
	}

	runner, err = pm.RunFactory(extCmd, pm.NewContainerProcess, onpid, onexit)
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
	if ps, ok := ps.(pm.ContainerProcess); !ok {
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
		pm.System("btrfs", "subvolume", "delete", path.Join(BackendBaseDir, c.name()))
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
