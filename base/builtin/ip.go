package builtin

import (
	"encoding/json"
	"fmt"
	"github.com/vishvananda/netlink"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"net"
)

func init() {
	pm.CmdMap["ip.bridge.create"] = process.NewInternalProcessFactory(brCreate)
	pm.CmdMap["ip.bridge.delete"] = process.NewInternalProcessFactory(brDelete)
	pm.CmdMap["ip.bridge.addif"] = process.NewInternalProcessFactory(brAddInf)
	pm.CmdMap["ip.bridge.delif"] = process.NewInternalProcessFactory(brDelInf)

	pm.CmdMap["ip.link.up"] = process.NewInternalProcessFactory(linkUp)
	pm.CmdMap["ip.link.down"] = process.NewInternalProcessFactory(linkDown)
}

type LinkArguments struct {
	Name string `json:"name"`
}

type BridgeArguments struct {
	LinkArguments
	HwAddress string `json:"hwaddr"`
}

func brCreate(cmd *core.Command) (interface{}, error) {
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

func brDelete(cmd *core.Command) (interface{}, error) {
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

func brAddInf(cmd *core.Command) (interface{}, error) {
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

func brDelInf(cmd *core.Command) (interface{}, error) {
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

func linkUp(cmd *core.Command) (interface{}, error) {
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

func linkDown(cmd *core.Command) (interface{}, error) {
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
