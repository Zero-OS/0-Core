package containers

import (
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"os"
	"path"
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
	id        uint16
	mgr       *containerManager
	route     core.Route
	Arguments ContainerCreateArguments `json:"arguments"`
	Root      string                   `json:"root"`
	pid       int
}

func newContainer(mgr *containerManager, id uint16, route core.Route, args ContainerCreateArguments) *container {
	c := &container{
		mgr:       mgr,
		id:        id,
		route:     route,
		Arguments: args,
	}
	c.Root = c.root()
	return c
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

	mgr := pm.GetManager()
	extCmd := &core.Command{
		ID:    coreID,
		Route: c.route,
		Arguments: core.MustArguments(
			process.ContainerCommandArguments{
				Name:        "/coreX",
				Chroot:      c.root(),
				Dir:         "/",
				HostNetwork: c.Arguments.HostNetwork,
				Args: []string{
					"-core-id", fmt.Sprintf("%d", c.id),
					"-redis-socket", "/redis.socket",
					"-reply-to", coreXResponseQueue,
					"-hostname", c.Arguments.Hostname,
				},
				Env: map[string]string{
					"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
					"HOME": "/",
				},
			},
		),
	}

	onpid := &pm.PIDHook{
		Action: c.onpid,
	}

	onexit := &pm.ExitHook{
		Action: c.onexit,
	}

	_, err = mgr.NewRunner(extCmd, process.NewContainerProcess, onpid, onexit)
	if err != nil {
		log.Errorf("error in container runner: %s", err)
		return
	}

	return
}

func (c *container) preStart() error {
	if c.Arguments.HostNetwork {
		return c.preStartHostNetworking()
	}

	if err := c.preStartIsolatedNetworking(); err != nil {
		return err
	}

	return nil
}

func (c *container) onpid(pid int) {
	c.pid = pid
	if err := c.postStart(); err != nil {
		log.Errorf("Container post start error: %s", err)
		//TODO. Should we shut the container down?
	}
}

func (c *container) onexit(state bool) {
	log.Debugf("Container %v exited with state %v", c.id, state)
	c.cleanup()
}

func (c *container) cleanup() {
	log.Debugf("cleaning up container-%d", c.id)
	defer c.mgr.cleanup(c.id)

	c.destroyNetwork()

	if err := c.unMountAll(); err != nil {
		log.Errorf("unmounting container-%d was not clean", err)
	}

	os.RemoveAll(path.Join(BackendBaseDir, c.name()))
	os.RemoveAll(c.root())
}

func (c *container) namespace() error {
	sourceNs := fmt.Sprintf("/proc/%d/ns/net", c.pid)
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
	if c.Arguments.HostNetwork {
		return nil
	}

	if err := c.postStartIsolatedNetworking(); err != nil {
		log.Errorf("isolated networking error: %s", err)
		return err
	}

	return nil
}
