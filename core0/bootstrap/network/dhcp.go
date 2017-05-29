package network

import (
	"fmt"
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
				Args: []string{"-i", inf, "-s", "/usr/share/udhcp/simple.script", "-q"},
			},
		),
		MaxTime: 5,
	}

	runner, err := pm.GetManager().RunCmd(cmd)

	if err != nil {
		return err
	}

	result := runner.Wait()

	if result == nil || result.State != core.StateSuccess {
		return fmt.Errorf("dhcpcd failed on interface %s: (%s) %s", inf, result.State, result.Streams)
	}

	return nil
}
