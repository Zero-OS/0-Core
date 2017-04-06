package containers

import (
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/pborman/uuid"
	"github.com/vishvananda/netlink"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"syscall"
)

var (
	devicesToBind = []string{"random", "urandom", "null"}
)

type container struct {
	id    uint16
	route core.Route
	args  *ContainerCreateArguments

	pid int
}

func newContainer(id uint16, route core.Route, args *ContainerCreateArguments) *container {
	return &container{
		id:    id,
		route: route,
		args:  args,
	}
}

func (c *container) Start() error {
	coreID := fmt.Sprintf("core-%d", c.id)

	if err := c.mount(); err != nil {
		c.cleanup()
		log.Errorf("error in container mount: %s", err)
		return err
	}

	if err := c.preStart(); err != nil {
		c.cleanup()
		log.Errorf("error in container prestart: %s", err)
		return err
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
				HostNetwork: c.args.HostNetwork,
				Args: []string{
					"-core-id", fmt.Sprintf("%d", c.id),
					"-redis-socket", "/redis.socket",
					"-reply-to", coreXResponseQueue,
					"-hostname", c.args.Hostname,
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

	_, err := mgr.NewRunner(extCmd, process.NewContainerProcess, onpid, onexit)
	if err != nil {
		c.cleanup()
		log.Errorf("error in container runner: %s", err)
		return err
	}

	return nil
}

func (c *container) preStartHostNetworking() error {
	os.MkdirAll(path.Join(c.root(), "etc"), 0755)
	p := path.Join(c.root(), "etc", "resolv.conf")
	os.Remove(p)
	ioutil.WriteFile(p, []byte{}, 0644) //touch the file.
	return syscall.Mount("/etc/resolv.conf", p, "", syscall.MS_BIND, "")
}

func (c *container) preStart() error {
	//mount up redis socket, coreX binary, etc...
	root := c.root()

	redisSocketTarget := path.Join(root, "redis.socket")
	coreXTarget := path.Join(root, coreXBinaryName)

	if f, err := os.Create(redisSocketTarget); err == nil {
		f.Close()
	} else {
		log.Errorf("Failed to touch file '%s': %s", redisSocketTarget, err)
	}

	if f, err := os.Create(coreXTarget); err == nil {
		f.Close()
	} else {
		log.Errorf("Failed to touch file '%s': %s", coreXTarget, err)
	}

	if err := syscall.Mount(redisSocketSrc, redisSocketTarget, "", syscall.MS_BIND, ""); err != nil {
		return err
	}

	coreXSrc, err := exec.LookPath(coreXBinaryName)
	if err != nil {
		return err
	}

	if err := syscall.Mount(coreXSrc, coreXTarget, "", syscall.MS_BIND, ""); err != nil {
		return err
	}

	if c.args.HostNetwork {
		return c.preStartHostNetworking()
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

	if !c.args.HostNetwork {
		c.unPortForward()
		//remove bridge links
		//TODO: unbridging here.
		//for _, bridge := range c.args.Network.Bridge {
		//	c.unbridge(bridge)
		//}

		pm.GetManager().Kill(fmt.Sprintf("net-%v", c.id))

		if c.pid > 0 {
			targetNs := fmt.Sprintf("/run/netns/%v", c.id)

			if err := syscall.Unmount(targetNs, 0); err != nil {
				log.Errorf("Failed to unmount %s: %s", targetNs, err)
			}
			os.RemoveAll(targetNs)
		}
	}

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

func (c *container) zerotier(netID string) error {
	args := map[string]interface{}{
		"netns":    c.id,
		"zerotier": netID,
	}

	netcmd := core.Command{
		ID:        fmt.Sprintf("net-%v", c.id),
		Command:   zeroTierCommand,
		Arguments: core.MustArguments(args),
	}

	_, err := pm.GetManager().RunCmd(&netcmd)
	return err
}

func (c *container) unbridge(bridge ContainerBridgeSettings) error {
	name := fmt.Sprintf("%s-%v", bridge.Name(), c.id)

	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}

	return netlink.LinkDel(link)
}

func (c *container) bridge(index int, bridge ContainerBridgeSettings) error {
	link, err := netlink.LinkByName(bridge.Name())
	if err != nil {
		return err
	}

	if link.Type() != "bridge" {
		return fmt.Errorf("'%s' is not a bridge", link.Attrs().Name)
	}

	name := fmt.Sprintf("%s-%v", bridge.Name(), c.id)
	peerName := fmt.Sprintf("%s-%v-eth%d", bridge.Name(), c.id, index)

	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:        name,
			Flags:       net.FlagUp,
			MTU:         1500,
			TxQLen:      1000,
			MasterIndex: link.Attrs().Index,
		},
		PeerName: peerName,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("create link: %s", err)
	}

	peer, err := netlink.LinkByName(peerName)
	if err != nil {
		return fmt.Errorf("get peer: %s", err)
	}

	if err := netlink.LinkSetUp(peer); err != nil {
		return fmt.Errorf("set up: %s", err)
	}

	if err := netlink.LinkSetNsPid(peer, c.pid); err != nil {
		return fmt.Errorf("set ns pid: %s", err)
	}

	//TODO: this doesn't work after moving the device to the NS.
	//But we can't rename as well before joining the ns, otherwise we
	//can end up with conflicting name on the host namespace.
	//if err := netlink.LinkSetName(peer, fmt.Sprintf("eth%d", index)); err != nil {
	//	return fmt.Errorf("set link name: %s", err)
	//}

	dev := fmt.Sprintf("eth%d", index)

	cmd := &core.Command{
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: "ip",
				Args: []string{"netns", "exec", fmt.Sprintf("%v", c.id), "ip", "link", "set", peerName, "name", dev},
			},
		),
	}
	runner, err := pm.GetManager().RunCmd(cmd)

	if err != nil {
		return err
	}

	result := runner.Wait()
	if result.State != core.StateSuccess {
		return fmt.Errorf("failed to rename device: %s", result.Streams)
	}

	//setting up bridged networking
	switch bridge.Setup() {
	case "":
	case "none":
	case "dhcp":
		//start a dhcpc inside the container.
		dhcpc := &core.Command{
			ID:      uuid.New(),
			Command: process.CommandSystem,
			Arguments: core.MustArguments(
				process.SystemCommandArguments{
					Name: "ip",
					Args: []string{
						"netns",
						"exec",
						fmt.Sprintf("%v", c.id),
						"udhcpc", "-q", "-i", dev, "-s", "/usr/share/udhcp/simple.script",
					},
				},
			),
		}
		pm.GetManager().RunCmd(dhcpc)
	default:
		//set static ip
		if _, _, err := net.ParseCIDR(bridge.Setup()); err != nil {
			return err
		}

		{
			//putting the interface up
			cmd := &core.Command{
				ID:      uuid.New(),
				Command: process.CommandSystem,
				Arguments: core.MustArguments(
					process.SystemCommandArguments{
						Name: "ip",
						Args: []string{
							"netns",
							"exec",
							fmt.Sprintf("%v", c.id),
							"ip", "link", "set", "dev", dev, "up"},
					},
				),
			}

			runner, err := pm.GetManager().RunCmd(cmd)
			if err != nil {
				return err
			}
			result := runner.Wait()
			if result.State != core.StateSuccess {
				return fmt.Errorf("error brinding interface up: %v", result.Streams)
			}
		}

		{
			//setting the ip address
			cmd := &core.Command{
				ID:      uuid.New(),
				Command: process.CommandSystem,
				Arguments: core.MustArguments(
					process.SystemCommandArguments{
						Name: "ip",
						Args: []string{"netns", "exec", fmt.Sprintf("%v", c.id), "ip", "address", "add", bridge.Setup(), "dev", dev},
					},
				),
			}

			runner, err := pm.GetManager().RunCmd(cmd)
			if err != nil {
				return err
			}
			result := runner.Wait()
			if result.State != core.StateSuccess {
				return fmt.Errorf("error settings interface ip: %v", result.Streams)
			}
		}
	}
	return nil
}

func (c *container) getDefaultIP() net.IP {
	base := c.id + 1
	//we increment the ID to avoid getting the ip of the bridge itself.
	return net.IPv4(BridgeIP[0], BridgeIP[1], byte(base&0xff00>>8), byte(base&0x00ff))
}

func (c *container) setDefaultGateway(idx int) error {
	////setting the ip address
	eth := fmt.Sprintf("eth%d", idx)
	cmd := &core.Command{
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: "ip",
				Args: []string{"netns", "exec", fmt.Sprintf("%v", c.id),
					"ip", "route", "add", "metric", "1000", "default", "via", DefaultBridgeIP, "dev", eth},
			},
		),
	}

	runner, err := pm.GetManager().RunCmd(cmd)
	if err != nil {
		return err
	}

	result := runner.Wait()
	if result.State != core.StateSuccess {
		return fmt.Errorf("error settings interface ip: %v", result.Streams)
	}
	return nil
}

func (c *container) setDefaultDNS() error {
	file, err := os.OpenFile(path.Join(c.root(), "etc", "resolv.conf"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.WriteString(fmt.Sprintf("\nnameserver %s\n", DefaultBridgeIP))

	return err
}

func (c *container) forwardId(host int, container int) string {
	return fmt.Sprintf("socat-%d-%d-%d", c.id, host, container)
}

func (c *container) unPortForward() {
	for host, container := range c.args.Port {
		pm.GetManager().Kill(c.forwardId(host, container))
	}
}

func (c *container) setPortForwards() error {
	ip := c.getDefaultIP()

	for host, container := range c.args.Port {
		//nft add rule nat prerouting iif eth0 tcp dport { 80, 443 } dnat 192.168.1.120
		cmd := &core.Command{
			ID:      c.forwardId(host, container),
			Command: process.CommandSystem,
			Arguments: core.MustArguments(
				process.SystemCommandArguments{
					Name: "socat",
					Args: []string{
						fmt.Sprintf("tcp-listen:%d,reuseaddr,fork", host),
						fmt.Sprintf("tcp-connect:%s:%d", ip, container),
					},
					NoOutput: true,
				},
			),
		}

		onExit := &pm.ExitHook{
			Action: func(s bool) {
				log.Infof("Port forward %d:%d container: %d exited", host, container, c.id)
			},
		}

		pm.GetManager().RunCmd(cmd, onExit)
	}

	return nil
}

func (c *container) setDefaultNetwork(i int, net *Network) error {
	//Add to the default bridge
	brdige := ContainerBridgeSettings{
		DefaultBridgeName,
		fmt.Sprintf("%s/16", c.getDefaultIP()),
	}

	if err := c.bridge(i, brdige); err != nil {
		return err
	}

	//set default gateway
	if err := c.setDefaultGateway(i); err != nil {
		log.Errorf("Failed to set default gateway: %", err)
	}

	//set nameserver.
	if err := c.setDefaultDNS(); err != nil {
		log.Errorf("Failed to set default nameserver: %s", err)
	}

	if err := c.setPortForwards(); err != nil {
		log.Errorf("Failed to setup port forwarding: %s", err)
	}

	return nil
}

func (c *container) postStartIsolatedNetworking() error {
	//only setup networking if host-network is false
	if err := c.namespace(); err != nil {
		return err
	}

	for idx, network := range c.args.Network {
		switch network.Type {
		case "vxlan":
			//TODO: ensure vxlan, and get the bridge name
		case "vlan":
			//TODO: ensure vlan, and get the bridge name
		case "zerotier":
			if err := c.zerotier(network.ID); err != nil {
				return err
			}
		case "default":
			if err := c.setDefaultNetwork(idx, &network); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *container) postStart() error {
	if c.args.HostNetwork {
		return nil
	}

	if err := c.postStartIsolatedNetworking(); err != nil {
		log.Errorf("isolated networking error: %s", err)
		return err
	}

	return nil
}
