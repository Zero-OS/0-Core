package builtin

import (
	"encoding/json"
	"fmt"
	"github.com/vishvananda/netlink"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"github.com/zero-os/0-core/base/utils"
	"io/ioutil"
	"net"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"syscall"
)

type bridgeMgr struct {
	m sync.Mutex
}

func init() {
	b := &bridgeMgr{}
	pm.CmdMap["bridge.create"] = process.NewInternalProcessFactory(b.create)
	pm.CmdMap["bridge.list"] = process.NewInternalProcessFactory(b.list)
	pm.CmdMap["bridge.delete"] = process.NewInternalProcessFactory(b.delete)
	pm.CmdMap["bridge.add_host"] = process.NewInternalProcessFactory(b.addHost)
}

var (
	ruleHandlerP = regexp.MustCompile(`(?m:ip saddr ([\d\./]+) masquerade # handle (\d+))`)
	HandlerP     = regexp.MustCompile(`handle (\d+)$`)
)

const (
	NoneBridgeNetworkMode    BridgeNetworkMode = ""
	DnsMasqBridgeNetworkMode BridgeNetworkMode = "dnsmasq"
	StaticBridgeNetworkMode  BridgeNetworkMode = "static"
)

type BridgeNetworkMode string

type NetworkStaticSettings struct {
	CIDR string `json:"cidr"`
}

func (n *NetworkStaticSettings) Validate() error {
	ip, network, err := net.ParseCIDR(n.CIDR)
	if err != nil {
		return err
	}

	if network.IP.Equal(ip) {
		return fmt.Errorf("Invalid IP")
	}

	return nil
}

type NetworkDnsMasqSettings struct {
	NetworkStaticSettings
	Start net.IP `json:"start"`
	End   net.IP `json:"end"`
}

func (n *NetworkDnsMasqSettings) Validate() error {
	ip, network, err := net.ParseCIDR(n.CIDR)
	if err != nil {
		return err
	}

	if network.IP.Equal(ip) {
		return fmt.Errorf("Invalid IP")
	}

	if !network.Contains(n.Start) {
		return fmt.Errorf("start ip address out of range")
	}

	if !network.Contains(n.End) {
		return fmt.Errorf("end ip address out of range")
	}

	return nil
}

type BridgeNetwork struct {
	Mode     BridgeNetworkMode `json:"mode"`
	Nat      bool              `json:"nat"`
	Settings json.RawMessage   `json:"settings"`
}

type BridgeCreateArguments struct {
	Name      string        `json:"name"`
	HwAddress string        `json:"hwaddr"`
	Network   BridgeNetwork `json:"network"`
}

func (br *BridgeCreateArguments) Validate() error {
	name := len(br.Name)
	if 1 > name || name > 15 {
		return fmt.Errorf("Bridge name must be between 1 and 15 characters")
	}

	if br.Name == "default" {
		return fmt.Errorf("Bridge name can't be 'default'")
	}

	return nil
}

type BridgeDeleteArguments struct {
	Name string `json:"name"`
}

type BridgeAddHost struct {
	Bridge string `json:"bridge"`
	IP     string `json:"ip"`
	Mac    string `json:"mac"`
}

func (b *bridgeMgr) intersect(n1 *net.IPNet, n2 *net.IPNet) bool {
	ip1 := n1.IP.Mask(n2.Mask)
	ip2 := n2.IP.Mask(n1.Mask)
	return n2.Contains(ip1) || n1.Contains(ip2)
}

func (b *bridgeMgr) conflict(addr *netlink.Addr) error {
	links, err := netlink.LinkList()
	if err != nil {
		return err
	}
	for _, link := range links {
		addresses, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			return err
		}
		for _, address := range addresses {
			if b.intersect(address.IPNet, addr.IPNet) {
				return fmt.Errorf("overlapping range with %s on %s", address, link.Attrs().Name)
			}
		}
	}

	return nil
}

func (b *bridgeMgr) bridgeStaticNetworking(bridge *netlink.Bridge, network *BridgeNetwork) (*netlink.Addr, error) {
	var settings NetworkStaticSettings
	if err := json.Unmarshal(network.Settings, &settings); err != nil {
		return nil, err
	}

	if err := settings.Validate(); err != nil {
		return nil, err
	}

	addr, err := netlink.ParseAddr(settings.CIDR)
	if err != nil {
		return nil, err
	}

	if err := b.conflict(addr); err != nil {
		return nil, err
	}

	if err := netlink.AddrAdd(bridge, addr); err != nil {
		return nil, err
	}

	//we still dnsmasq also for the default bridge for dns resolving.

	leases := fmt.Sprintf("/var/lib/misc/%s.leases", bridge.Name)
	os.RemoveAll(leases)

	args := []string{
		"--no-hosts",
		"--keep-in-foreground",
		fmt.Sprintf("--pid-file=/var/run/dnsmasq/%s.pid", bridge.Name),
		fmt.Sprintf("--dhcp-leasefile=%s", leases),
		fmt.Sprintf("--listen-address=%s", addr.IP),
		fmt.Sprintf("--interface=%s", bridge.Name),
		"--bind-interfaces",
		"--except-interface=lo",
	}

	cmd := &core.Command{
		ID:      b.dnsmasqPName(bridge.Name),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: "dnsmasq",
				Args: args,
			},
		),
	}

	onExit := &pm.ExitHook{
		Action: func(state bool) {
			if !state {
				log.Errorf("dnsmasq for %s exited with an error", bridge.Name)
			}
		},
	}

	log.Debugf("dnsmasq(%s): %s", bridge.Name, args)
	_, err = pm.GetManager().RunCmd(cmd, onExit)

	if err != nil {
		return nil, err
	}

	return addr, nil
}

func (b *bridgeMgr) dnsmasqPName(n string) string {
	return fmt.Sprintf("dnsmasq-%s", n)
}

func (b *bridgeMgr) dnsmasqHostsFilePath(n string) string {
	return fmt.Sprintf("/var/run/dnsmasq/%s", b.dnsmasqPName(n))
}

func (b *bridgeMgr) bridgeDnsMasqNetworking(bridge *netlink.Bridge, network *BridgeNetwork) (*netlink.Addr, error) {
	var settings NetworkDnsMasqSettings
	if err := json.Unmarshal(network.Settings, &settings); err != nil {
		return nil, err
	}

	if err := settings.Validate(); err != nil {
		return nil, err
	}

	os.MkdirAll("/var/run/dnsmasq", 0755)

	addr, err := netlink.ParseAddr(settings.CIDR)
	if err != nil {
		return nil, err
	}

	if err := b.conflict(addr); err != nil {
		return nil, err
	}

	if err := netlink.AddrAdd(bridge, addr); err != nil {
		return nil, err
	}

	hostsFile := b.dnsmasqHostsFilePath(bridge.Name)
	os.RemoveAll(hostsFile)
	os.MkdirAll(hostsFile, 0755)

	leases := fmt.Sprintf("/var/lib/misc/%s.leases", bridge.Name)
	os.RemoveAll(leases)

	args := []string{
		"--no-hosts",
		"--keep-in-foreground",
		fmt.Sprintf("--pid-file=/var/run/dnsmasq/%s.pid", bridge.Name),
		fmt.Sprintf("--dhcp-leasefile=%s", leases),
		fmt.Sprintf("--listen-address=%s", addr.IP),
		fmt.Sprintf("--interface=%s", bridge.Name),
		fmt.Sprintf("--dhcp-range=%s,%s,%s", settings.Start, settings.End, net.IP(addr.Mask)),
		fmt.Sprintf("--dhcp-option=6,%s", addr.IP),
		fmt.Sprintf("--dhcp-hostsfile=%s", hostsFile),
		"--bind-interfaces",
		"--except-interface=lo",
	}

	cmd := &core.Command{
		ID:      b.dnsmasqPName(bridge.Name),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name:     "dnsmasq",
				Args:     args,
				NoOutput: true,
			},
		),
	}

	onExit := &pm.ExitHook{
		Action: func(state bool) {
			if !state {
				log.Errorf("dnsmasq for %s exited with an error", bridge.Name)
			}
		},
	}

	log.Debugf("dnsmasq(%s): %s", bridge.Name, args)
	_, err = pm.GetManager().RunCmd(cmd, onExit)

	if err != nil {
		return nil, err
	}

	return addr, nil
}

func (b *bridgeMgr) addHost(cmd *core.Command) (interface{}, error) {
	var args BridgeAddHost
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	name := b.dnsmasqPName(args.Bridge)
	runner, ok := pm.GetManager().Runner(name)
	if !ok {
		//either no bridge with that name, or this bridge does't have dnsmasq settings.
		return nil, fmt.Errorf("not supported no dnsmasq process found")
	}

	//write file for the host
	if err := ioutil.WriteFile(
		path.Join(b.dnsmasqHostsFilePath(args.Bridge), args.IP),
		[]byte(fmt.Sprintf("%s,%s", args.Mac, args.IP)),
		0644,
	); err != nil {
		return nil, err
	}

	if ps, ok := runner.Process().(process.Signaler); ok {
		if err := ps.Signal(syscall.SIGHUP); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (b *bridgeMgr) bridgeNetworking(bridge *netlink.Bridge, network *BridgeNetwork) error {
	var addr *netlink.Addr
	var err error
	switch network.Mode {
	case StaticBridgeNetworkMode:
		addr, err = b.bridgeStaticNetworking(bridge, network)
	case DnsMasqBridgeNetworkMode:
		addr, err = b.bridgeDnsMasqNetworking(bridge, network)
	case NoneBridgeNetworkMode:
		return nil
	default:
		return fmt.Errorf("invalid networking mode %s", network.Mode)
	}

	if err != nil {
		return err
	}

	if network.Nat && addr != nil {
		return b.setNAT(addr)
	}

	return nil
}

func (b *bridgeMgr) setNAT(addr *netlink.Addr) error {
	//enable nat-ting
	_, err := pm.GetManager().System("nft", "add", "rule", "nat", "post", "ip",
		"saddr", addr.IPNet.String(), "masquerade")

	return err
}

func (b *bridgeMgr) unsetNAT(addr []netlink.Addr) error {
	//enable nat-ting
	job, err := pm.GetManager().System("nft", "list", "ruleset", "-a")
	if err != nil {
		return err
	}

	var ips []string
	for _, ip := range addr {
		//this trick to get the corred network ID from netlink addresses
		_, n, _ := net.ParseCIDR(ip.IPNet.String())
		ips = append(ips, n.String())
	}

	for _, line := range ruleHandlerP.FindAllStringSubmatch(job.Streams.Stdout(), -1) {
		ip := line[1]
		handle := line[2]
		if utils.InString(ips, ip) {
			pm.GetManager().System("nft", "delete", "rule", "nat", "post", "handle", handle)
		}
	}

	return nil
}

func (b *bridgeMgr) create(cmd *core.Command) (interface{}, error) {
	b.m.Lock()
	defer b.m.Unlock()

	var args BridgeCreateArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	if err := args.Validate(); err != nil {
		return nil, err
	}

	var hw net.HardwareAddr

	if args.HwAddress != "" {
		var err error
		hw, err = net.ParseMAC(args.HwAddress)
		if err != nil {
			return nil, err
		}
	}

	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name:         args.Name,
			HardwareAddr: hw,
			TxQLen:       1000, //needed other wise bridge won't work
		},
	}

	if err := netlink.LinkAdd(bridge); err != nil {
		return nil, err
	}

	var err error

	defer func() {
		if err != nil {
			netlink.LinkDel(bridge)
		}
	}()

	if args.HwAddress != "" {
		if err = netlink.LinkSetHardwareAddr(bridge, hw); err != nil {
			return nil, err
		}
	}

	if err = netlink.LinkSetUp(bridge); err != nil {
		return nil, err
	}

	if err := b.isolate(args.Name); err != nil {
		return nil, err
	}

	if err = b.bridgeNetworking(bridge, &args.Network); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *bridgeMgr) isolate(br string) error {
	name := fmt.Sprintf("\"%v\"", br)
	//make sure all packages coming from this interface is marked with (1)
	if _, err := pm.GetManager().System("nft",
		"add", "rule", "nat", "pre", "iifname", name, "meta", "mark", "set", "1"); err != nil {
		return err
	}

	//any packages that is coming and exiting on the same bridge should be marked as 2
	if _, err := pm.GetManager().System("nft",
		"add", "rule", "filter", "forward", "iifname", name, "oifname", name, "meta", "mark", "set", "2"); err != nil {
		return err
	}

	//drop any package that is forwarded to this bridge and still marked as 1
	if _, err := pm.GetManager().System("nft",
		"add", "rule", "filter", "forward", "oifname", name, "meta", "mark", "1", "drop"); err != nil {
		return err
	}

	return nil
}

func (b *bridgeMgr) unIsolate(br string) {
	job, err := pm.GetManager().System("nft", "list", "ruleset", "-a")
	if err != nil {
		log.Errorf("failed to list nft rules: %s", err)
		return
	}

	var table, chain string
	for _, line := range strings.Split(job.Streams.Stdout(), "\n") {
		line = strings.TrimSpace(line)
		fmt.Sscanf(line, "table ip %s {", &table)
		fmt.Sscanf(line, "chain %s {", &chain)

		if strings.Index(line, fmt.Sprintf(`"%s"`, br)) == -1 {
			continue
		}

		//get handler.
		m := HandlerP.FindStringSubmatch(line)
		if len(m) != 2 {
			continue
		}
		_, err := pm.GetManager().System("nft", "delete", "rule", table, chain, "handle", m[1])
		if err != nil {
			log.Errorf("failed to remove nft rule '%s/%s/%s'", table, chain, m[1])
		}
	}

}

func (b *bridgeMgr) list(cmd *core.Command) (interface{}, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	bridges := make([]string, 0)
	for _, link := range links {
		if link.Type() == "bridge" {
			bridges = append(bridges, link.Attrs().Name)
		}
	}

	return bridges, nil
}

func (b *bridgeMgr) delete(cmd *core.Command) (interface{}, error) {
	b.m.Lock()
	b.m.Unlock()

	var args BridgeDeleteArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	link, err := netlink.LinkByName(args.Name)
	if err != nil {
		return nil, err
	}

	if link.Type() != "bridge" {
		return nil, fmt.Errorf("bridge not found")
	}

	//make sure to stop dnsmasq, just in case it's running
	pm.GetManager().Kill(fmt.Sprintf("dnsmasq-%s", link.Attrs().Name))

	addresses, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return nil, err
	}

	b.unIsolate(args.Name)

	if err := b.unsetNAT(addresses); err != nil {
		return nil, err
	}

	if err := netlink.LinkDel(link); err != nil {
		return nil, err
	}

	return nil, nil
}
