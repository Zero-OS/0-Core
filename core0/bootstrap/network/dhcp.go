package network

import (
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
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
	cmd := &core.Command{
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: "udhcpc",
				Args: []string{"-f", "-i", inf, "-A", "3", "-s", "/usr/share/udhcp/simple.script"},
			},
		),
	}

	_, err := pm.GetManager().RunCmd(cmd)

	return err
}
