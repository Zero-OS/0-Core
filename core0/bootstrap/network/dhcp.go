package network

import (
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
)

const (
	ProtocolDHCP = "dhcp"
)

func init() {
	protocols[ProtocolDHCP] = &dhcpProtocol{}
}

type dhcpProtocol struct {
}

func (d *dhcpProtocol) Configure(mgr NetworkManager, inf string) error {
	result, err := pm.GetManager().System("udhcpc", "-i", inf, "-s", "/usr/share/udhcp/simple.script", "-q")
	if err != nil {
		return err
	}

	if result == nil || result.State != core.StateSuccess {
		return fmt.Errorf("dhcpcd failed on interface %s: (%s) %s", inf, result.State, result.Streams)
	}

	return nil
}
