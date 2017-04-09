package network

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"net"
)

const (
	ProtocolStatic = "static"
)

type StaticProtocol interface {
	Protocol
	ConfigureStatic(ip *net.IPNet, inf string) error
}

func init() {
	protocols[ProtocolStatic] = &staticProtocol{}
}

type staticProtocol struct{}

func (s *staticProtocol) ConfigureStatic(ip *net.IPNet, inf string) error {
	log.Debugf("Configure '%s' with static ip '%s'", inf, ip)

	link, err := netlink.LinkByName(inf)
	if err != nil {
		return err
	}

	addr := &netlink.Addr{IPNet: ip}
	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return err
	}

	exists := false
	for _, a := range addrs {
		if a.IPNet.String() == addr.IPNet.String() {
			exists = true
			break
		}
	}

	if !exists {
		if err := netlink.AddrAdd(link, addr); err != nil {
			return err
		}
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return err
	}

	return nil
}

func (s *staticProtocol) Configure(mgr NetworkManager, inf string) error {
	link, err := netlink.LinkByName(inf)
	if err != nil {
		return err
	}

	setting, ok := mgr.getConfig().Static[inf]
	if !ok {
		return fmt.Errorf("missing static configuration for interface '%s'", inf)
	}

	ip, err := netlink.ParseIPNet(setting.IP)

	if err := s.ConfigureStatic(ip, inf); err != nil {
		return err
	}

	if setting.Gateway == "" {
		return nil
	}

	//setting up gateway.
	rip := net.ParseIP(setting.Gateway)
	if rip == nil {
		return fmt.Errorf("invalid ip for gateway '%s'", rip)
	}

	route := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Src:       ip.IP,
		Gw:        rip,
	}

	return netlink.RouteAdd(route)
}
