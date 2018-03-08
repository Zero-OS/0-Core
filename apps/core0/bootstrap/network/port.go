package network

import (
	"fmt"
	"github.com/vishvananda/netlink"
)

func PortCreate(n string) error {
	a := netlink.NewLinkAttrs()
	a.Name = n
	t := netlink.Tuntap{
		LinkAttrs: a,
		Mode:      netlink.TUNTAP_MODE_TAP,
	}

	links, err := netlink.LinkList()
	if err != nil {
		return err
	}

	for _, link := range links {
		if link.Attrs().Name == n {
			if _, ok := link.(*netlink.Tuntap); ok {
				//device is already there. just return
				return nil
			} else {
				return fmt.Errorf("conflicting device name")
			}
		}
	}

	return netlink.LinkAdd(&t)
}

func DeviceUp(n string) error {
	l, e := netlink.LinkByName(n)
	if e != nil {
		return e
	}

	return netlink.LinkSetUp(l)
}
