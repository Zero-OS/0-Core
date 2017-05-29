package containers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"github.com/pborman/uuid"
	"github.com/vishvananda/netlink"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strconv"
	"syscall"
	"time"
)

const (
	containerLinkNameFmt = "cont%d-%d"
	containerPeerNameFmt = "%sp"
)

func (c *container) preStartHostNetworking() error {
	os.MkdirAll(path.Join(c.root(), "etc"), 0755)
	p := path.Join(c.root(), "etc", "resolv.conf")
	os.Remove(p)
	ioutil.WriteFile(p, []byte{}, 0644) //touch the file.
	return syscall.Mount("/etc/resolv.conf", p, "", syscall.MS_BIND, "")
}

func (c *container) zerotierHome() string {
	return fmt.Sprintf("/tmp/zerotier/container-%d", c.id)
}

func (c *container) zerotierDaemon() error {
	c.zto.Do(func() {
		home := c.zerotierHome()
		os.RemoveAll(home)
		os.MkdirAll(home, 0755)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		hook := &pm.PIDHook{
			Action: func(_ int) {
				log.Info("checking for zt availability")
				var err error
				for i := 0; i < 10; i++ {
					_, err = pm.GetManager().System("ip", "netns", "exec", fmt.Sprint(c.ID()), "zerotier-cli", fmt.Sprintf("-D%s", home), "listnetworks")
					if err == nil {
						break
					}
					<-time.After(1 * time.Second)
				}

				if err != nil {
					c.zterr = fmt.Errorf("daemon couldn't start: %s", err)
				}

				cancel()
			},
		}

		exit := &pm.ExitHook{
			Action: func(s bool) {
				c.zterr = fmt.Errorf("zerotier for container '%d' exited with '%v'", c.ID(), s)
				cancel()
			},
		}

		c.zt, c.zterr = pm.GetManager().RunCmd(&core.Command{
			ID:      uuid.New(),
			Command: process.CommandSystem,
			Arguments: core.MustArguments(
				process.SystemCommandArguments{
					Name: "ip",
					Args: []string{
						"netns", "exec", fmt.Sprint(c.ID()), "zerotier-one", "-p0", home,
					},
				},
			),
		}, hook, exit)

		if c.zterr != nil {
			return
		}

		//wait for it to start
		select {
		case <-ctx.Done():
		case <-time.After(120 * time.Second):
			c.zterr = fmt.Errorf("timedout waiting for zt daemon to start")
		}
	})

	return c.zterr
}

type ztNetorkInfo struct {
	PortDeviceName    string   `json:"portDeviceName"`
	AssignedAddresses []string `json:"assignedAddresses"`
	NetID             string   `json:"nwid"`
}

func (c *container) postZerotierNetwork(idx int, netID string) error {
	if err := c.zerotierDaemon(); err != nil {
		return err
	}

	home := c.zerotierHome()
	_, err := pm.GetManager().System("ip", "netns", "exec", fmt.Sprint(c.ID()), "zerotier-cli", fmt.Sprintf("-D%s", home), "join", netID)
	return err
}

func (c *container) postBridge(dev string, index int, n *Nic) error {
	name := fmt.Sprintf(containerLinkNameFmt, c.id, index)
	peerName := fmt.Sprintf(containerPeerNameFmt, name)

	peer, err := netlink.LinkByName(peerName)
	if err != nil {
		return fmt.Errorf("get peer: %s", err)
	}

	if err := netlink.LinkSetUp(peer); err != nil {
		return fmt.Errorf("set peer up: %s", err)
	}

	if err := netlink.LinkSetNsPid(peer, c.PID); err != nil {
		return fmt.Errorf("set ns pid: %s", err)
	}

	//TODO: this doesn't work after moving the device to the NS.
	//But we can't rename as well before joining the ns, otherwise we
	//can end up with conflicting name on the host namespace.
	//if err := netlink.LinkSetName(peer, fmt.Sprintf("eth%d", index)); err != nil {
	//	return fmt.Errorf("set link name: %s", err)
	//}

	_, err = pm.GetManager().System("ip", "netns", "exec", fmt.Sprintf("%v", c.id), "ip", "link", "set", peerName, "name", dev)
	if err != nil {
		return fmt.Errorf("failed to rename device: %s", err)
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
					Env: map[string]string{
						"ROOT": c.root(),
					},
				},
			),
		}
		pm.GetManager().RunCmd(dhcpc)
	} else if n.Config.CIDR != "" {
		if _, _, err := net.ParseCIDR(n.Config.CIDR); err != nil {
			return err
		}

		//putting the interface up
		_, err := pm.GetManager().System("ip", "netns",
			"exec",
			fmt.Sprintf("%v", c.id),
			"ip", "link", "set", "dev", dev, "up")

		if err != nil {
			return fmt.Errorf("error brinding interface up: %v", err)
		}

		//setting the ip address
		_, err = pm.GetManager().System("ip", "netns", "exec", fmt.Sprintf("%v", c.id), "ip", "address", "add", n.Config.CIDR, "dev", dev)
		if err != nil {
			return fmt.Errorf("error settings interface ip: %v", err)
		}
	}

	if n.Config.Gateway != "" {
		if err := c.setGateway(dev, n.Config.Gateway); err != nil {
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

func (c *container) preBridge(index int, bridge string, n *Nic, ovs Container) error {
	link, err := netlink.LinkByName(bridge)
	if err != nil {
		return fmt.Errorf("bridge '%s' not found: %s", bridge, err)
	}

	name := fmt.Sprintf(containerLinkNameFmt, c.id, index)
	peerName := fmt.Sprintf(containerPeerNameFmt, name)

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
		result, err := c.mgr.Dispatch(ovs.ID(), &core.Command{
			Command: "ovs.port-add",
			Arguments: core.MustArguments(
				map[string]interface{}{
					"bridge": bridge,
					"port":   name,
				},
			),
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
	for host, container := range c.Args.Port {
		pm.GetManager().Kill(c.forwardId(host, container))
	}
}

func (c *container) setPortForwards() error {
	ip := c.getDefaultIP()

	for host, container := range c.Args.Port {
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

func (c *container) setGateway(dev string, gw string) error {
	////setting the ip address
	_, err := pm.GetManager().System("ip", "netns", "exec", fmt.Sprintf("%v", c.id),
		"ip", "route", "add", "metric", "1000", "default", "via", gw, "dev", dev)

	if err != nil {
		return fmt.Errorf("error settings interface ip: %v", err)
	}

	return nil
}

func (c *container) postDefaultNetwork(name string, idx int, net *Nic) error {
	//Add to the default bridge
	defnet := &Nic{
		Config: NetworkConfig{
			CIDR:    fmt.Sprintf("%s/16", c.getDefaultIP().String()),
			Gateway: DefaultBridgeIP,
			DNS:     []string{DefaultBridgeIP},
		},
	}

	if err := c.postBridge(name, idx, defnet); err != nil {
		return err
	}

	if err := c.setPortForwards(); err != nil {
		return err
	}

	return nil
}

func (c *container) preDefaultNetwork(i int, net *Nic) error {
	//Add to the default bridge

	defnet := &Nic{
		Config: NetworkConfig{
			CIDR:    fmt.Sprintf("%s/16", c.getDefaultIP().String()),
			Gateway: DefaultBridgeIP,
			DNS:     []string{DefaultBridgeIP},
		},
	}

	if err := c.preBridge(i, DefaultBridgeName, defnet, nil); err != nil {
		return err
	}

	return nil
}

func (c *container) preVxlanNetwork(idx int, net *Nic) error {
	vxlan, err := strconv.ParseInt(net.ID, 10, 64)
	if err != nil {
		return err
	}
	//find the container with OVS tag
	ovs := c.mgr.GetOneWithTags(OVSTag)
	if ovs == nil {
		return fmt.Errorf("ovs is needed for VXLAN network type")
	}

	//ensure that a bridge is available with that vlan tag.
	//we dispatch the ovs.vlan-ensure command to container.
	result, err := c.mgr.Dispatch(ovs.ID(), &core.Command{
		Command: "ovs.vxlan-ensure",
		Arguments: core.MustArguments(map[string]interface{}{
			"master": OVSVXBackend,
			"vxlan":  vxlan,
		}),
	})

	if err != nil {
		return err
	}

	if result.State != core.StateSuccess {
		return fmt.Errorf("failed to ensure vxlan bridge: %v", result.Data)
	}

	var bridge string
	if err := json.Unmarshal([]byte(result.Data), &bridge); err != nil {
		return fmt.Errorf("failed to load vxlan-ensure result: %s", err)
	}
	log.Debugf("vxlan bridge name: %d", bridge)
	//we have the vxlan bridge name
	return c.preBridge(idx, bridge, net, ovs)
}

func (c *container) postVxlanNetwork(name string, idx int, net *Nic) error {
	//we have the vxlan bridge name
	return c.postBridge(name, idx, net)
}

func (c *container) preVlanNetwork(idx int, net *Nic) error {
	vlanID, err := strconv.ParseInt(net.ID, 10, 16)
	if err != nil {
		return err
	}
	if vlanID < 0 || vlanID >= 4095 {
		return fmt.Errorf("invalid vlan id (0-4094)")
	}
	//find the container with OVS tag

	ovs := c.mgr.GetOneWithTags(OVSTag)
	if ovs == nil {
		return fmt.Errorf("ovs is needed for VLAN network type")
	}

	//ensure that a bridge is available with that vlan tag.
	//we dispatch the ovs.vlan-ensure command to container.
	result, err := c.mgr.Dispatch(ovs.ID(), &core.Command{
		Command: "ovs.vlan-ensure",
		Arguments: core.MustArguments(map[string]interface{}{
			"master": OVSBackPlane,
			"vlan":   vlanID,
		}),
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
	return c.preBridge(idx, bridge, net, ovs)
}

func (c *container) postVlanNetwork(name string, idx int, net *Nic) error {
	return c.postBridge(name, idx, net)
}

func (c *container) postStartIsolatedNetworking() error {
	if err := c.namespace(); err != nil {
		return err
	}

	for idx, network := range c.Args.Nics {
		var err error
		var name string
		name = fmt.Sprintf("eth%d", idx)
		if network.Name != "" {
			name = network.Name
		}

		switch network.Type {
		case "vxlan":
			err = c.postVxlanNetwork(name, idx, &network)
		case "vlan":
			err = c.postVlanNetwork(name, idx, &network)
		case "zerotier":
			err = c.postZerotierNetwork(idx, network.ID)
		case "default":
			err = c.postDefaultNetwork(name, idx, &network)
		case "bridge":
			err = c.postBridge(name, idx, &network)
		}

		if err != nil {
			log.Errorf("failed to initialize network '%v': %s", network, err)
		}
	}

	return nil
}

func (c *container) preStartIsolatedNetworking() error {
	for idx, network := range c.Args.Nics {
		switch network.Type {
		case "vxlan":
			if err := c.preVxlanNetwork(idx, &network); err != nil {
				return err
			}
		case "vlan":
			if err := c.preVlanNetwork(idx, &network); err != nil {
				return err
			}
		case "default":
			if err := c.preDefaultNetwork(idx, &network); err != nil {
				return err
			}
		case "bridge":
			if err := c.preBridge(idx, network.ID, &network, nil); err != nil {
				return err
			}
		case "zerotier":
		default:
			return fmt.Errorf("unkown network type '%s'", network.Type)
		}
	}

	return nil
}

func (c *container) unBridge(idx int, n *Nic, ovs Container) {
	name := fmt.Sprintf(containerLinkNameFmt, c.id, idx)
	if ovs != nil {
		_, err := c.mgr.Dispatch(ovs.ID(), &core.Command{
			Command: "ovs.port-del",
			Arguments: core.MustArguments(map[string]interface{}{
				"port": name,
			}),
		})

		if err != nil {
			log.Errorf("failed to delete port %s: %s", name, err)
		}
		return
	}

	link, err := netlink.LinkByName(name)
	if err != nil {
		return
	}

	netlink.LinkDel(link)
}

func (c *container) destroyNetwork() {
	log.Debugf("destroying networking for container: %s", c.id)
	if c.Args.HostNetwork {
		//nothing to do.
		return
	}

	for idx, network := range c.Args.Nics {
		switch network.Type {
		case "vxlan":
			fallthrough
		case "vlan":
			ovs := c.mgr.GetOneWithTags(OVSTag)
			c.unBridge(idx, &network, ovs)
		case "default":
			c.unBridge(idx, &network, nil)
			c.unPortForward()
		}
	}

	if c.zt != nil {
		c.zt.Terminate()
	}

	//clean up namespace
	if c.PID > 0 {
		targetNs := fmt.Sprintf("/run/netns/%v", c.id)

		if err := syscall.Unmount(targetNs, 0); err != nil {
			log.Errorf("Failed to unmount %s: %s", targetNs, err)
		}
		os.RemoveAll(targetNs)
	}
}
