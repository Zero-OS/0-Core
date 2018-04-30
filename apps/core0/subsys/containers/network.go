package containers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pborman/uuid"
	"github.com/vishvananda/netlink"
	"github.com/zero-os/0-core/apps/core0/helper/socat"
	"github.com/zero-os/0-core/base/pm"
)

const (
	containerLinkNameFmt          = "cont%d-%d"
	containerMonitoredLinkNameFmt = "contm%d-%d"
	containerPeerNameFmt          = "%sp"
)

func (c *container) preStartHostNetworking() error {
	os.MkdirAll(path.Join(c.root(), "etc"), 0755)
	p := path.Join(c.root(), "etc", "resolv.conf")
	os.Remove(p)
	ioutil.WriteFile(p, []byte{}, 0644) //touch the file.
	return syscall.Mount("/etc/resolv.conf", p, "", syscall.MS_BIND, "")
}

func (c *container) zerotierHome() string {
	return path.Join(BackendBaseDir, c.name(), "zerotier")
}

func (c *container) zerotierID() string {
	return fmt.Sprintf("container-%d-zerotier", c.id)
}

func (c *container) startZerotier() (pm.Job, error) {
	home := c.zerotierHome()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var err error
	hook := &pm.PIDHook{
		Action: func(_ int) {
			log.Info("checking for zt availability")
			for i := 0; i < 10; i++ {
				_, err = pm.System("ip", "netns", "exec", fmt.Sprint(c.ID()), "zerotier-cli", fmt.Sprintf("-D%s", home), "listnetworks")
				if err == nil {
					break
				}
				<-time.After(1 * time.Second)
			}

			cancel()
		},
	}

	exit := &pm.ExitHook{
		Action: func(s bool) {
			err = fmt.Errorf("zerotier for container '%d' exited with '%v'", c.ID(), s)
			cancel()
		},
	}

	var job pm.Job
	job, err = pm.Run(&pm.Command{
		ID:      c.zerotierID(),
		Command: pm.CommandSystem,
		Arguments: pm.MustArguments(
			pm.SystemCommandArguments{
				Name: "ip",
				Args: []string{
					"netns", "exec", fmt.Sprint(c.ID()), "zerotier-one", "-p0", home,
				},
			},
		),
	}, hook, exit)

	if err != nil {
		return nil, err
	}

	//wait for it to start
	select {
	case <-ctx.Done():
	case <-time.After(120 * time.Second):
		return nil, fmt.Errorf("timedout waiting for zt daemon to start")
	}

	return job, nil
}

func (c *container) watchZerotier(job pm.Job) {
	for {
		job.Wait()
		if c.terminating {
			return
		}

		var err error
		job, err = c.startZerotier()
		if err != nil {
			log.Errorf("failed to restart zerotier: %s", err)
			c.zterr = err
			<-time.After(1 * time.Second)
		}
	}
}

func (c *container) zerotierDaemon() error {
	c.zto.Do(func() {
		home := c.zerotierHome()
		os.RemoveAll(home)
		os.MkdirAll(home, 0755)

		if len(c.Args.Identity) > 0 {
			//set zt identity
			if err := ioutil.WriteFile(path.Join(c.zerotierHome(), "identity.secret"), []byte(c.Args.Identity), 0600); err != nil {
				log.Errorf("failed to write zerotier secret identity: %v", err)
			}
			parts := strings.Split(c.Args.Identity, ":")
			public := strings.Join(parts[:len(parts)-1], ":")
			if err := ioutil.WriteFile(path.Join(c.zerotierHome(), "identity.public"), []byte(public), 0644); err != nil {
				log.Errorf("failed to write zerotier public identity: %v", err)
			}
		}

		var job pm.Job
		job, c.zterr = c.startZerotier()
		if c.zterr != nil {
			log.Errorf("error while starting zerotier daemon for container: %d (%s): re-spawning", c.id, c.zterr)
			job.Signal(syscall.SIGTERM)
		}
		//start the watcher anyway
		go c.watchZerotier(job)
	})

	return c.zterr
}

type ztNetorkInfo struct {
	PortDeviceName    string   `json:"portDeviceName"`
	AssignedAddresses []string `json:"assignedAddresses"`
	NetID             string   `json:"nwid"`
}

func (c *container) joinZerotierNetwork(idx int, netID string) error {
	if err := c.zerotierDaemon(); err != nil {
		return err
	}

	home := c.zerotierHome()
	_, err := pm.System("ip", "netns", "exec", fmt.Sprint(c.ID()), "zerotier-cli", fmt.Sprintf("-D%s", home), "join", netID)
	return err
}

func (c *container) leaveZerotierNetwork(idx int, netID string) error {
	if err := c.zerotierDaemon(); err != nil {
		return err
	}

	home := c.zerotierHome()
	_, err := pm.System("ip", "netns", "exec", fmt.Sprint(c.ID()), "zerotier-cli", fmt.Sprintf("-D%s", home), "leave", netID)
	return err
}

func (c *container) setupLink(src, target string, index int, n *Nic) error {

	link, err := netlink.LinkByName(src)
	if err != nil {
		return fmt.Errorf("get link: %s", err)
	}

	if err := netlink.LinkSetNsPid(link, c.PID); err != nil {
		return fmt.Errorf("set ns pid: %s", err)
	}

	//TODO: this doesn't work after moving the device to the NS.
	//But we can't rename as well before joining the ns, otherwise we
	//can end up with conflicting name on the host namespace.
	//if err := netlink.LinkSetName(peer, fmt.Sprintf("eth%d", index)); err != nil {
	//	return fmt.Errorf("set link name: %s", err)
	//}

	_, err = pm.System("ip", "netns", "exec", fmt.Sprintf("%v", c.id), "ip", "link", "set", src, "name", target)
	if err != nil {
		return fmt.Errorf("failed to rename device: %s", err)
	}

	if n.Config.Dhcp {
		//start a dhcpc inside the container.
		dhcpc := &pm.Command{
			ID:      uuid.New(),
			Command: pm.CommandSystem,
			Arguments: pm.MustArguments(
				pm.SystemCommandArguments{
					Name: "ip",
					Args: []string{
						"netns",
						"exec",
						fmt.Sprintf("%v", c.id),
						"udhcpc", "-q", "-i", target, "-s", "/usr/share/udhcp/simple.script",
					},
					Env: map[string]string{
						"ROOT": c.root(),
					},
				},
			),
		}
		pm.Run(dhcpc)
	} else if n.Config.CIDR != "" {
		if _, _, err := net.ParseCIDR(n.Config.CIDR); err != nil {
			return err
		}

		//putting the interface up
		_, err := pm.System("ip", "netns",
			"exec",
			fmt.Sprintf("%v", c.id),
			"ip", "link", "set", "dev", target, "up")

		if err != nil {
			return fmt.Errorf("error bringing interface up: %v", err)
		}

		//setting the ip address
		_, err = pm.System("ip", "netns", "exec", fmt.Sprintf("%v", c.id), "ip", "address", "add", n.Config.CIDR, "dev", target)
		if err != nil {
			return fmt.Errorf("error settings interface ip: %v", err)
		}
	}

	if n.Config.Gateway != "" {
		if err := c.setGateway(target, n.Config.Gateway); err != nil {
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

func (c *container) postBridge(dev string, index int, n *Nic) error {
	var name string
	if n.Monitor {
		name = fmt.Sprintf(containerMonitoredLinkNameFmt, c.id, index)
	} else {
		name = fmt.Sprintf(containerLinkNameFmt, c.id, index)
	}
	peerName := fmt.Sprintf(containerPeerNameFmt, name)

	return c.setupLink(peerName, dev, index, n)
}

func (c *container) postLink(dev string, index int, n *Nic) error {
	var name string
	if n.Monitor {
		name = fmt.Sprintf(containerMonitoredLinkNameFmt, c.id, index)
	} else {
		name = fmt.Sprintf(containerLinkNameFmt, c.id, index)
	}

	return c.setupLink(name, dev, index, n)
}

func (c *container) preBridge(index int, bridge string, n *Nic, ovs Container) error {
	link, err := netlink.LinkByName(bridge)
	if err != nil {
		return pm.NotFoundError(fmt.Errorf("bridge '%s' not found: %s", bridge, err))
	}

	var name string
	if n.Monitor {
		name = fmt.Sprintf(containerMonitoredLinkNameFmt, c.id, index)
	} else {
		name = fmt.Sprintf(containerLinkNameFmt, c.id, index)
	}
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
			return pm.BadRequestError(fmt.Errorf("'%s' is not a bridge", bridge))
		}
		br := link.(*netlink.Bridge)
		if err := netlink.LinkSetMaster(veth, br); err != nil {
			return err
		}
	} else {
		//with ovs
		result, err := c.mgr.Dispatch(ovs.ID(), &pm.Command{
			Command: "ovs.port-add",
			Arguments: pm.MustArguments(
				map[string]interface{}{
					"bridge": bridge,
					"port":   name,
				},
			),
		})

		if err != nil {
			return fmt.Errorf("ovs dispatch error: %s", err)
		}

		if result.State != pm.StateSuccess {
			return fmt.Errorf("failed to attach veth to bridge: %s", result.Data)
		}
	}

	peer, err := netlink.LinkByName(peerName)
	if err != nil {
		return fmt.Errorf("get peer: %s", err)
	}

	if n.HWAddress != "" {
		if mac, err := net.ParseMAC(n.HWAddress); err != nil {
			log.Errorf("parse hwaddr error: %s", err)
		} else if err := netlink.LinkSetHardwareAddr(peer, mac); err != nil {
			return fmt.Errorf("failed to setup hw address: %s", err)
		}
	}

	return nil
}

func (c *container) preMacVlanNetwork(index int, n *Nic) error {
	link, err := netlink.LinkByName(n.ID)
	if err != nil {
		return pm.NotFoundError(fmt.Errorf("link '%s' not found: %s", n.ID, err))
	}

	var name string
	if n.Monitor {
		name = fmt.Sprintf(containerMonitoredLinkNameFmt, c.id, index)
	} else {
		name = fmt.Sprintf(containerLinkNameFmt, c.id, index)
	}

	macVlan := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        name,
			ParentIndex: link.Attrs().Index,
			Flags:       net.FlagUp,
			MTU:         1500,
			TxQLen:      1000,
		},
		Mode: netlink.MACVLAN_MODE_BRIDGE,
	}

	if err := netlink.LinkAdd(macVlan); err != nil {
		return fmt.Errorf("create macvlan link fail: %s", err)
	}

	if n.HWAddress != "" {
		if mac, err := net.ParseMAC(n.HWAddress); err != nil {
			log.Errorf("parse hwaddr error: %s", err)
		} else if err := netlink.LinkSetHardwareAddr(macVlan, mac); err != nil {
			return fmt.Errorf("failed to setup hw address: %s", err)
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

func (c *container) forwardId() string {
	return fmt.Sprintf("container-%d", c.id)
}

func (c *container) setPortForward(host string, dest int) error {
	ip := c.getDefaultIP().String()
	return socat.SetPortForward(c.forwardId(), ip, host, dest)
}

func (c *container) setPortForwards() error {
	for host, dest := range c.Args.Port {
		if err := c.setPortForward(host, dest); err != nil {
			return err
		}
	}

	return nil
}

func (c *container) setGateway(dev string, gw string) error {
	////setting the ip address
	_, err := pm.System("ip", "netns", "exec", fmt.Sprintf("%v", c.id),
		"ip", "route", "add", "metric", "1000", "default", "via", gw, "dev", dev)

	if err != nil {
		return fmt.Errorf("error settings default gateway: %v", err)
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
		return pm.PreconditionFailedError(fmt.Errorf("ovs is needed for VXLAN network type"))
	}

	//ensure that a bridge is available with that vlan tag.
	//we dispatch the ovs.vlan-ensure command to container.
	result, err := c.mgr.Dispatch(ovs.ID(), &pm.Command{
		Command: "ovs.vxlan-ensure",
		Arguments: pm.MustArguments(map[string]interface{}{
			"master": OVSVXBackend,
			"vxlan":  vxlan,
		}),
	})

	if err != nil {
		return err
	}

	if result.State != pm.StateSuccess {
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
		return pm.PreconditionFailedError(fmt.Errorf("ovs is needed for VLAN network type"))
	}

	//ensure that a bridge is available with that vlan tag.
	//we dispatch the ovs.vlan-ensure command to container.
	result, err := c.mgr.Dispatch(ovs.ID(), &pm.Command{
		Command: "ovs.vlan-ensure",
		Arguments: pm.MustArguments(map[string]interface{}{
			"master": OVSBackPlane,
			"vlan":   vlanID,
		}),
	})

	if err != nil {
		return err
	}

	if result.State != pm.StateSuccess {
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

func (c *container) postStartNetwork(idx int, network *Nic) (err error) {
	var name string
	name = fmt.Sprintf("eth%d", idx)
	if network.Name != "" {
		name = network.Name
	}

	switch network.Type {
	case "vxlan":
		err = c.postVxlanNetwork(name, idx, network)
	case "vlan":
		err = c.postVlanNetwork(name, idx, network)
	case "zerotier":
		err = c.joinZerotierNetwork(idx, network.ID)
	case "default":
		err = c.postDefaultNetwork(name, idx, network)
	case "bridge":
		err = c.postBridge(name, idx, network)
	case "macvlan":
		err = c.postLink(name, idx, network)
	}

	if err != nil {
		network.State = NicStateError
	} else {
		network.State = NicStateConfigured
	}
	return
}

func (c *container) postStartIsolatedNetworking() error {
	if err := c.namespace(); err != nil {
		return err
	}

	for idx, network := range c.Args.Nics {
		if err := c.postStartNetwork(idx, network); err != nil {
			log.Errorf("failed to initialize network '%v': %s", network, err)
		}
	}

	return nil
}

func (c *container) preStartNetwork(idx int, network *Nic) (err error) {
	network.State = NicStateUnknown
	switch network.Type {
	case "vxlan":
		err = c.preVxlanNetwork(idx, network)
	case "vlan":
		err = c.preVlanNetwork(idx, network)
	case "default":
		err = c.preDefaultNetwork(idx, network)
	case "bridge":
		err = c.preBridge(idx, network.ID, network, nil)
	case "macvlan":
		err = c.preMacVlanNetwork(idx, network)
	case "zerotier":
	default:
		err = pm.BadRequestError(fmt.Errorf("unkown network type '%s'", network.Type))
	}

	if err != nil {
		network.State = NicStateError
	}

	return
}

func (c *container) preStartIsolatedNetworking() error {
	for idx, network := range c.Args.Nics {
		if err := c.preStartNetwork(idx, network); err != nil {
			return err
		}
	}

	return nil
}

func (c *container) unLink(idx int, n *Nic) error {
	if n.Type != "macvlan" {
		return fmt.Errorf("unlink is only for macvlan nic type")
	}

	name := fmt.Sprintf("eth%d", idx)
	if _, err := pm.System("ip", "netns", "exec", fmt.Sprint(c.id), "ip", "link", "del", name); err != nil {
		return err
	}

	n.State = NicStateDestroyed
	return nil
}

func (c *container) unBridge(idx int, n *Nic, ovs Container) error {
	var name string
	if n.Monitor {
		name = fmt.Sprintf(containerMonitoredLinkNameFmt, c.id, idx)
	} else {
		name = fmt.Sprintf(containerLinkNameFmt, c.id, idx)
	}
	n.State = NicStateDestroyed
	if ovs != nil {
		_, err := c.mgr.Dispatch(ovs.ID(), &pm.Command{
			Command: "ovs.port-del",
			Arguments: pm.MustArguments(map[string]interface{}{
				"port": name,
			}),
		})

		if err != nil {
			return err
		}
	}

	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}

	return netlink.LinkDel(link)
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
			c.unBridge(idx, network, ovs)
		case "default":
			c.unBridge(idx, network, nil)
			socat.RemoveAll(c.forwardId())
		}
	}

	pm.Kill(c.zerotierID())

	//clean up namespace
	if c.PID > 0 {
		targetNs := fmt.Sprintf("/run/netns/%v", c.id)

		if err := syscall.Unmount(targetNs, 0); err != nil {
			log.Errorf("Failed to unmount %s: %s", targetNs, err)
		}
		os.RemoveAll(targetNs)
	}
}
