package containers

import (
	"encoding/json"
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/pborman/uuid"
	"io"
	"os"
	"path"
	"sync"
	"syscall"
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

	ch Channel
}

func newContainer(mgr *containerManager, id uint16, route core.Route, args ContainerCreateArguments) *container {
	c := &container{
		mgr:   mgr,
		id:    id,
		route: route,
		Args:  args,
	}
	c.Root = c.root()
	return c
}

func (c *container) ID() uint16 {
	return c.id
}

func (c *container) Arguments() ContainerCreateArguments {
	return c.Args
}

func (c *container) exec(bin string, args ...string) (pm.Runner, error) {
	return pm.GetManager().RunCmd(&core.Command{
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: bin,
				Args: args,
			},
		),
	})
}

func (c *container) sync(bin string, args ...string) (*core.JobResult, error) {
	runner, err := c.exec(bin, args...)
	if err != nil {
		return nil, err
	}

	job := runner.Wait()
	if job.State != core.StateSuccess {
		return nil, fmt.Errorf("%s exited with error (%s): %v", job.State, job.Streams)
	}

	return job, nil
}

func (c *container) Start() (err error) {
	coreID := fmt.Sprintf("core-%d", c.id)

	var local, remote Channel
	local, remote, err = Pipe()
	if err != nil {
		return err
	}

	c.ch = local
	defer func() {
		if err != nil {
			remote.Close()
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
				Files:       []uintptr{remote.Reader(), remote.Writer()},
				Env: map[string]string{
					"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
					"HOME": "/",
				},
			},
		),
	}

	onpid := &pm.PIDHook{
		Action: func(pid int) {
			remote.Close()
			c.onpid(pid)
		},
	}

	onexit := &pm.ExitHook{
		Action: c.onexit,
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

func (c *container) onpid(pid int) {
	c.PID = pid
	if !c.Args.Privileged {
		c.mgr.cgroup.Task(pid)
	}

	if err := c.postStart(); err != nil {
		log.Errorf("Container post start error: %s", err)
		//TODO. Should we shut the container down?
	}

	go c.communicate()
}

func (c *container) communicate() {
	decoder := json.NewDecoder(c.ch)
	for {
		var result core.JobResult
		err := decoder.Decode(&result)
		if err == io.EOF {
			return
		} else if err != nil {
			log.Errorf("failed to process corex message: %s", err)
			continue
		}

		c.mgr.sink.Forward(&result)
	}
}

func (c *container) onexit(state bool) {
	log.Debugf("Container %v exited with state %v", c.id, state)
	c.cleanup()
}

func (c *container) cleanup() {
	log.Debugf("cleaning up container-%d", c.id)
	defer c.mgr.unset_container(c.id)

	c.ch.Close()
	c.destroyNetwork()

	if err := c.unMountAll(); err != nil {
		log.Errorf("unmounting container-%d was not clean", err)
	}
}

func (c *container) cleanSandbox() {
	if c.getFSType(BackendBaseDir) == "btrfs" {
		c.sync("btrfs", "subvolume", "delete", path.Join(BackendBaseDir, c.name()))
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
