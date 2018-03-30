package network

import (
	"fmt"
	"github.com/pborman/uuid"
	"github.com/zero-os/0-core/base/pm"
	"io/ioutil"
)

const (
	ProtocolDHCP = "dhcp"
)

func init() {
	protocols[ProtocolDHCP] = &dhcpProtocol{}
}

type dhcpProtocol struct {
}

func (d *dhcpProtocol) getZerotierId() (string, error) {
	bytes, err := ioutil.ReadFile("/tmp/zt/identity.public")
	if err != nil {
		return "", err
	}

	return string(bytes)[0:10], nil
}

func dhcpCommand(inf string, hostid string, persistant bool) *pm.Command {
	cmd := &pm.Command{
		ID:        uuid.New(),
		Command:   pm.CommandSystem,
		Arguments: nil,
	}

	if persistant {
		cmd.Arguments = pm.MustArguments(
			pm.SystemCommandArguments{
				Name: "udhcpc",
				Args: []string{
					"-b", // background
					"-i", inf,
					"-t", "10", // try 10 times before giving up
					"-A", "3", // wait 3 seconds between each trial
					"--now", // exit if failed after consuming all the trials (otherwise stay alive)
					"-s", "/usr/share/udhcp/simple.script",
					"-x", hostid, // set hostname on dhcp request
				},
			},
		)

	} else {
		cmd.Arguments = pm.MustArguments(
			pm.SystemCommandArguments{
				Name: "udhcpc",
				Args: []string{
					"-f", // foreground
					"-i", inf,
					"-t", "10", // try 10 times before giving up
					"-A", "3", // wait 3 seconds between each trial
					"--now",  // exit if failed after consuming all the trials (otherwise stay alive)
					"--quit", // exit if lease was obtained
					"-s", "/usr/share/udhcp/simple.script",
					"-x", hostid, // set hostname on dhcp request
				},
			},
		)
	}

	return cmd
}

func (d *dhcpProtocol) Configure(mgr NetworkManager, inf string) error {
	hostid := "hostname:zero-os"

	ztid, err := d.getZerotierId()
	if err == nil {
		hostid = fmt.Sprintf("hostname:zero-os-%s", ztid)
	}

	cmd := dhcpCommand(inf, hostid, false)
	job, err := pm.Run(cmd)
	if err != nil {
		return err
	}

	result := job.Wait()
	if result.State != pm.StateSuccess {
		return fmt.Errorf("udhcpc failed: %s", result.Streams.Stderr())
	}

	cmd = dhcpCommand(inf, hostid, true)
	job, err = pm.Run(cmd)
	if err != nil {
		return err
	}

	return nil
}
