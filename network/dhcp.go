package network

import (
	"fmt"
	"github.com/g8os/core.base/pm"
	"github.com/g8os/core.base/pm/core"
	"github.com/g8os/core.base/pm/process"
)

const (
	ProtocolDHCP = "dhcp"
)

func init() {
	protocols[ProtocolDHCP] = &dhcpProtocol{}
}

type DHCPProtocol interface {
	Protocol
	Stop(inf string)
}

type dhcpProtocol struct {
}

func (d *dhcpProtocol) Stop(inf string) {
	cmd := &core.Command{
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			map[string]interface{}{
				"name": "udhcpc",
				"args": []string{"-i", inf}, // FIXME
			},
		),
	}

	runner, err := pm.GetManager().RunCmd(cmd)
	if err == nil {
		runner.Wait()
	}
}

func (d *dhcpProtocol) Configure(mgr NetworkManager, inf string) error {
	d.Stop(inf)

	cmd := &core.Command{
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			map[string]interface{}{
				"name": "udhcpc",
				"args": []string{"-i", inf, "-s", "/usr/share/udhcp/simple.script", "-q"},
			},
		),
	}

	runner, err := pm.GetManager().RunCmd(cmd)

	if err != nil {
		return err
	}

	result := runner.Wait()

	if result == nil || result.State != core.StateSuccess {
		return fmt.Errorf("dhcpcd failed on interface %s: %s", inf, result.Streams)
	}

	return nil
}
