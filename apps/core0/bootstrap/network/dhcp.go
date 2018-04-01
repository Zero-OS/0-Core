package network

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/pborman/uuid"
	"github.com/zero-os/0-core/base/pm"
)

const (
	ProtocolDHCP = "dhcp"

	carrierFile = "/sys/class/net/%s/carrier"
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

func (d *dhcpProtocol) isPlugged(inf string) error {
	data, err := ioutil.ReadFile(fmt.Sprintf(carrierFile, inf))
	if err != nil {
		return err
	}
	data = bytes.TrimSpace(data)
	if string(data) == "1" {
		return nil
	}

	return fmt.Errorf("interface %s has no carrier(%s)", inf, string(data))
}

func (d *dhcpProtocol) Configure(mgr NetworkManager, inf string) error {
	if err := d.isPlugged(inf); err != nil {
		return err
	}

	hostid := "hostname:zero-os"

	ztid, err := d.getZerotierId()
	if err == nil {
		hostid = fmt.Sprintf("hostname:zero-os-%s", ztid)
	}

	cmd := &pm.Command{
		ID:      uuid.New(),
		Command: pm.CommandSystem,
		Arguments: pm.MustArguments(
			pm.SystemCommandArguments{
				Name: "udhcpc",
				Args: []string{
					"-f", //foreground
					"-i", inf,
					"-t", "10", //try 10 times before giving up
					"-A", "3", //wait 3 seconds between each trial
					"-s", "/usr/share/udhcp/simple.script",
					"-x", hostid, //set hostname on dhcp request
				},
			},
		),
	}

	_, err = pm.Run(cmd)
	if err != nil {
		return err
	}

	/*TODO:
	Should we wait until we have an IP?

	note we can't wait for the job itself, since the udhcpc client does not exit after optianing
	an IP anymore. Hence we can never tell if an IP was assigned unless we pull for the interface addresses
	before we give up.
	*/
	return nil
}
