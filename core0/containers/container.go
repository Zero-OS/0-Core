package containers

import (
	"encoding/json"
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
	"strconv"
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

	if c.Arguments.HostNetwork {
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
	defer c.mgr.cleanup(c.id)

	if !c.Arguments.HostNetwork {
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

//
//func (c *container) unbridge(bridge ContainerBridgeSettings) error {
//	name := fmt.Sprintf("%s-%v", bridge.Name(), c.id)
//
//	link, err := netlink.LinkByName(name)
//	if err != nil {
//		return err
//	}
//
//	return netlink.LinkDel(link)
//}

func (c *container) bridge(index int, bridge string, n *Network, ovs *container) error {
	link, err := netlink.LinkByName(bridge)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("cont%d-%d", c.id, index)
	peerName := fmt.Sprintf("%sp", name)

	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:   name,
			Flags:  net.FlagUp,
			MTU:    1500,
			TxQLen: 1000,
		},
		PeerName: peerName,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("create veth pair fail: %s", err)
	}

	//setting the master
	if ovs == nil {
		//no ovs
		if link.Type() != "bridge" {
			return fmt.Errorf("'%s' is not a bridge", bridge)
		}
		br := link.(*netlink.Bridge)
		if err := netlink.LinkSetMaster(veth, br); err != nil {
			return err
		}
	} else {
		//with ovs
		result, err := c.mgr.dispatchSync(&ContainerDispatchArguments{
			Container: ovs.id,
			Command: core.Command{
				Command: "ovs.port-add",
				Arguments: core.MustArguments(
					map[string]interface{}{
						"bridge": bridge,
						"port":   name,
					},
				),
			},
		})

		if err != nil {
			return fmt.Errorf("ovs dispatch error: %s", err)
		}

		if result.State != core.StateSuccess {
			return fmt.Errorf("failed to attach veth to bridge: %s", result.Data)
		}
	}

	peer, err := netlink.LinkByName(peerName)
	if err != nil {
		return fmt.Errorf("get peer: %s", err)
	}
	if n.HWAddress != "" {
		mac, err := net.ParseMAC(n.HWAddress)
		if err == nil {
			if err := netlink.LinkSetHardwareAddr(peer, mac); err != nil {
				return fmt.Errorf("failed to setup hw address: %s", err)
			}
		} else {
			log.Errorf("parse hwaddr error: %s", err)
		}
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

	if n.Config.Dhcp {
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
	} else if n.Config.CIDR != "" {
		if _, _, err := net.ParseCIDR(n.Config.CIDR); err != nil {
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
						Args: []string{"netns", "exec", fmt.Sprintf("%v", c.id), "ip", "address", "add", n.Config.CIDR, "dev", dev},
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

	if n.Config.Gateway != "" {
		if err := c.setGateway(index, n.Config.Gateway); err != nil {
			return err
		}
	}

	for _, dns := range n.Config.DNS {
		if err := c.setDNS(dns); err != nil {
			return err
		}
	}

	return nil
}

func (c *container) getDefaultIP() net.IP {
	base := c.id + 1
	//we increment the ID to avoid getting the ip of the bridge itself.
	return net.IPv4(BridgeIP[0], BridgeIP[1], byte(base&0xff00>>8), byte(base&0x00ff))
}

func (c *container) setDNS(dns string) error {
	file, err := os.OpenFile(path.Join(c.root(), "etc", "resolv.conf"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.WriteString(fmt.Sprintf("\nnameserver %s\n", dns))

	return err
}

func (c *container) forwardId(host int, container int) string {
	return fmt.Sprintf("socat-%d-%d-%d", c.id, host, container)
}

func (c *container) unPortForward() {
	for host, container := range c.Arguments.Port {
		pm.GetManager().Kill(c.forwardId(host, container))
	}
}

func (c *container) setPortForwards() error {
	ip := c.getDefaultIP()

	for host, container := range c.Arguments.Port {
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

func (c *container) setGateway(idx int, gw string) error {
	////setting the ip address
	eth := fmt.Sprintf("eth%d", idx)
	cmd := &core.Command{
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: "ip",
				Args: []string{"netns", "exec", fmt.Sprintf("%v", c.id),
					"ip", "route", "add", "metric", "1000", "default", "via", gw, "dev", eth},
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

func (c *container) setDefaultNetwork(i int, net *Network) error {
	//Add to the default bridge

	defnet := &Network{
		Config: NetworkConfig{
			CIDR:    fmt.Sprintf("%s/16", c.getDefaultIP().String()),
			Gateway: DefaultBridgeIP,
			DNS:     []string{DefaultBridgeIP},
		},
	}

	if err := c.bridge(i, DefaultBridgeName, defnet, nil); err != nil {
		return err
	}

	if err := c.setPortForwards(); err != nil {
		log.Errorf("Failed to setup port forwarding: %s", err)
	}

	return nil
}

func (c *container) vxlan(idx int, net *Network) error {
	vxlan, err := strconv.ParseInt(net.ID, 10, 64)
	if err != nil {
		return err
	}
	//find the container with OVS tag
	ovs := c.mgr.getOneWithTags(OVSTag)
	if ovs == nil {
		return fmt.Errorf("ovs is needed for VLAN network type")
	}

	//ensure that a bridge is available with that vlan tag.
	//we dispatch the ovs.vlan-ensure command to container.
	result, err := c.mgr.dispatchSync(&ContainerDispatchArguments{
		Container: ovs.id,
		Command: core.Command{
			Command: "ovs.vxlan-ensure",
			Arguments: core.MustArguments(map[string]interface{}{
				"master": OVSVXBackend,
				"vxlan":  vxlan,
			}),
		},
	})

	if err != nil {
		return err
	}

	if result.State != core.StateSuccess {
		return fmt.Errorf("failed to ensure vlan bridge: %v", result.Data)
	}
	//brname:
	var bridge string
	if err := json.Unmarshal([]byte(result.Data), &bridge); err != nil {
		return fmt.Errorf("failed to load vlan-ensure result: %s", err)
	}
	log.Debugf("vxlan bridge name: %d", bridge)
	//we have the vxlan bridge name
	return c.bridge(idx, bridge, net, ovs)
}

func (c *container) vlan(idx int, net *Network) error {
	vlanID, err := strconv.ParseInt(net.ID, 10, 16)
	if err != nil {
		return err
	}
	if vlanID == 0 || vlanID >= 4095 {
		return fmt.Errorf("invalid vlan id (1-4094)")
	}
	//find the container with OVS tag

	ovs := c.mgr.getOneWithTags(OVSTag)
	if ovs == nil {
		return fmt.Errorf("ovs is needed for VLAN network type")
	}

	//ensure that a bridge is available with that vlan tag.
	//we dispatch the ovs.vlan-ensure command to container.
	result, err := c.mgr.dispatchSync(&ContainerDispatchArguments{
		Container: ovs.id,
		Command: core.Command{
			Command: "ovs.vlan-ensure",
			Arguments: core.MustArguments(map[string]interface{}{
				"master": OVSBackPlane,
				"vlan":   vlanID,
			}),
		},
	})

	if err != nil {
		return err
	}

	if result.State != core.StateSuccess {
		return fmt.Errorf("failed to ensure vlan bridge: %v", result.Data)
	}
	//brname:
	var bridge string
	if err := json.Unmarshal([]byte(result.Data), &bridge); err != nil {
		return fmt.Errorf("failed to load vlan-ensure result: %s", err)
	}
	log.Debugf("vlan bridge name: %d", bridge)
	//we have the vlan bridge name
	return c.bridge(idx, bridge, net, ovs)
}

func (c *container) postStartIsolatedNetworking() error {
	if err := c.namespace(); err != nil {
		return err
	}

	for idx, network := range c.Arguments.Network {
		var err error
		switch network.Type {
		case "vxlan":
			err = c.vxlan(idx, &network)
		case "vlan":
			err = c.vlan(idx, &network)
		case "zerotier":
			//TODO: needs refactoring to support multiple
			//zerotier networks
			err = c.zerotier(network.ID)
		case "default":
			err = c.setDefaultNetwork(idx, &network)
		}

		if err != nil {
			log.Errorf("failed to initialize network '%v': %s", network, err)
		}
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
