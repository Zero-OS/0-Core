package builtin

import (
	"encoding/json"
	"fmt"
	"github.com/vishvananda/netlink"
	"github.com/zero-os/0-core/base/pm"
	"net"
)

type ipmgr struct{}

func init() {
	mgr := (*ipmgr)(nil)
	pm.RegisterBuiltIn("ip.bridge.add", mgr.brAdd)
	pm.RegisterBuiltIn("ip.bridge.del", mgr.brDel)
	pm.RegisterBuiltIn("ip.bridge.addif", mgr.brAddInf)
	pm.RegisterBuiltIn("ip.bridge.delif", mgr.brDelInf)

	pm.RegisterBuiltIn("ip.link.up", mgr.linkUp)
	pm.RegisterBuiltIn("ip.link.down", mgr.linkDown)
	pm.RegisterBuiltIn("ip.link.name", mgr.linkName)
	pm.RegisterBuiltIn("ip.link.list", mgr.linkList)

	pm.RegisterBuiltIn("ip.addr.add", mgr.addrAdd)
	pm.RegisterBuiltIn("ip.addr.del", mgr.addrDel)
	pm.RegisterBuiltIn("ip.addr.list", mgr.addrList)

	pm.RegisterBuiltIn("ip.route.add", mgr.routeAdd)
	pm.RegisterBuiltIn("ip.route.del", mgr.routeDel)
	pm.RegisterBuiltIn("ip.route.list", mgr.routeList)
}

type LinkArguments struct {
	Name string `json:"name"`
}

type BridgeArguments struct {
	LinkArguments
	HwAddress string `json:"hwaddr"`
}

func (_ *ipmgr) brAdd(cmd *pm.Command) (interface{}, error) {
	var args BridgeArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
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

	br := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name:   args.Name,
			TxQLen: 1000,
		},
	}

	if err := netlink.LinkAdd(br); err != nil {
		return nil, err
	}

	var err error
	defer func() {
		if err != nil {
			netlink.LinkDel(br)
		}
	}()

	if args.HwAddress != "" {
		err = netlink.LinkSetHardwareAddr(br, hw)
	}

	return nil, err
}

func (_ *ipmgr) brDel(cmd *pm.Command) (interface{}, error) {
	var args LinkArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	link, err := netlink.LinkByName(args.Name)
	if err != nil {
		return nil, err
	}
	if link.Type() != "bridge" {
		return nil, fmt.Errorf("no bridge with name '%s'", args.Name)
	}

	return nil, netlink.LinkDel(link)
}

type BridgeInfArguments struct {
	LinkArguments
	Inf string `json:"inf"`
}

func (_ *ipmgr) brAddInf(cmd *pm.Command) (interface{}, error) {
	var args BridgeInfArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	link, err := netlink.LinkByName(args.Name)
	if err != nil {
		return nil, err
	}
	if link.Type() != "bridge" {
		return nil, fmt.Errorf("no bridge with name '%s'", args.Name)
	}

	inf, err := netlink.LinkByName(args.Inf)
	if err != nil {
		return nil, err
	}

	return nil, netlink.LinkSetMaster(inf, link.(*netlink.Bridge))
}

func (_ *ipmgr) brDelInf(cmd *pm.Command) (interface{}, error) {
	var args BridgeInfArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	link, err := netlink.LinkByName(args.Name)
	if err != nil {
		return nil, err
	}
	if link.Type() != "bridge" {
		return nil, fmt.Errorf("no bridge with name '%s'", args.Name)
	}

	inf, err := netlink.LinkByName(args.Inf)
	if err != nil {
		return nil, err
	}

	if inf.Attrs().MasterIndex != link.Attrs().Index {
		return nil, fmt.Errorf("interface is not connected to bridge")
	}

	return nil, netlink.LinkSetNoMaster(inf)
}

func (_ *ipmgr) linkUp(cmd *pm.Command) (interface{}, error) {
	var args LinkArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	link, err := netlink.LinkByName(args.Name)
	if err != nil {
		return nil, err
	}

	return nil, netlink.LinkSetUp(link)
}

type LinkNameArguments struct {
	LinkArguments
	New string `json:"new"`
}

func (_ *ipmgr) linkName(cmd *pm.Command) (interface{}, error) {
	var args LinkNameArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	link, err := netlink.LinkByName(args.Name)
	if err != nil {
		return nil, err
	}

	return nil, netlink.LinkSetName(link, args.New)
}

func (_ *ipmgr) linkDown(cmd *pm.Command) (interface{}, error) {
	var args LinkArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	link, err := netlink.LinkByName(args.Name)
	if err != nil {
		return nil, err
	}

	return nil, netlink.LinkSetDown(link)
}

type Link struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	HwAddr string `json:"hwaddr"`
	Master string `json:"master"`
	Up     bool   `json:"up"`
}

func (_ *ipmgr) linkList(cmd *pm.Command) (interface{}, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	result := make([]Link, 0)
	for _, link := range links {
		master := ""
		if link.Attrs().MasterIndex != 0 {
			for _, l := range links {
				if link.Attrs().MasterIndex == l.Attrs().Index {
					master = l.Attrs().Name
				}
			}
		}

		result = append(result,
			Link{
				Type:   link.Type(),
				Name:   link.Attrs().Name,
				HwAddr: link.Attrs().HardwareAddr.String(),
				Master: master,
				Up:     link.Attrs().Flags&net.FlagUp != 0,
			},
		)
	}

	return result, nil
}

type AddrArguments struct {
	LinkArguments
	IP string `json:"ip"`
}

func (_ *ipmgr) addrAdd(cmd *pm.Command) (interface{}, error) {
	var args AddrArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	link, err := netlink.LinkByName(args.Name)
	if err != nil {
		return nil, err
	}

	addr, err := netlink.ParseAddr(args.IP)
	if err != nil {
		return nil, err
	}

	return nil, netlink.AddrAdd(link, addr)
}

func (_ *ipmgr) addrDel(cmd *pm.Command) (interface{}, error) {
	var args AddrArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	link, err := netlink.LinkByName(args.Name)
	if err != nil {
		return nil, err
	}

	addr, err := netlink.ParseAddr(args.IP)
	if err != nil {
		return nil, err
	}

	return nil, netlink.AddrDel(link, addr)
}

func (_ *ipmgr) addrList(cmd *pm.Command) (interface{}, error) {
	var args LinkArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	link, err := netlink.LinkByName(args.Name)
	if err != nil {
		return nil, err
	}

	addr, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return nil, err
	}

	ips := make([]string, 0)
	for _, addr := range addr {
		ips = append(ips, addr.IPNet.String())
	}

	return ips, err
}

type Route struct {
	Dev string `json:"dev"`
	Dst string `json:"dst"`
	Gw  string `json:"gw"`
}

func (r *Route) route() (*netlink.Route, error) {
	link, err := netlink.LinkByName(r.Dev)
	if err != nil {
		return nil, err
	}

	route := &netlink.Route{
		LinkIndex: link.Attrs().Index,
	}

	if r.Dst != "" {
		dst, err := netlink.ParseIPNet(r.Dst)
		if err != nil {
			return nil, err
		}
		route.Dst = dst
	}

	if r.Gw != "" {
		gw := net.ParseIP(r.Gw)
		if gw == nil {
			return nil, fmt.Errorf("invalid gw ip '%s'", r.Gw)
		}
		route.Gw = gw
	}

	return route, nil
}

func (_ *ipmgr) routeAdd(cmd *pm.Command) (interface{}, error) {
	var args Route
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	route, err := args.route()
	if err != nil {
		return nil, err
	}

	return nil, netlink.RouteAdd(route)
}

func (_ *ipmgr) routeDel(cmd *pm.Command) (interface{}, error) {
	var args Route
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	route, err := args.route()
	if err != nil {
		return nil, err
	}

	filter := netlink.RT_FILTER_OIF
	if route.Dst != nil {
		filter |= netlink.RT_FILTER_DST
	}
	if len(route.Gw) != 0 {
		filter |= netlink.RT_FILTER_GW
	}

	routes, err := netlink.RouteListFiltered(netlink.FAMILY_ALL, route, filter)
	if err != nil {
		return nil, err
	}

	if len(routes) == 0 {
		return nil, fmt.Errorf("route not found")
	} else if len(routes) > 1 {
		return nil, fmt.Errorf("ambiguous route matches multiple routes")
	}

	return nil, netlink.RouteDel(&routes[0])
}

func (_ *ipmgr) routeList(cmd *pm.Command) (interface{}, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return nil, err
	}

	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	results := make([]Route, 0)
	for _, r := range routes {
		var dst, gw, dev string
		for _, l := range links {
			if r.LinkIndex == l.Attrs().Index {
				dev = l.Attrs().Name
				break
			}
		}

		if r.Dst != nil {
			dst = r.Dst.String()
		}
		if r.Gw != nil {
			gw = r.Gw.String()
		}

		results = append(results,
			Route{
				Dst: dst,
				Gw:  gw,
				Dev: dev,
			},
		)
	}

	return results, nil
}
