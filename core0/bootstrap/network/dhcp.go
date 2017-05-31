package network

import (
	"github.com/pborman/uuid"
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
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: "udhcpc",
				Args: []string{
					"-f", //foreground
					"-i", inf,
					"-t", "10", //try 10 times before giving up
					"-A", "3", //wait 3 seconds between each trial
					"--now", //exit if failed after consuming all the trials (otherwise stay alive)
					"-s", "/usr/share/udhcp/simple.script"},
			},
		),
	}

	_, err := pm.GetManager().RunCmd(cmd)

	return err
}
