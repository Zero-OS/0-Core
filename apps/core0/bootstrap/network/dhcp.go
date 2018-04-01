package network

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"syscall"
	"time"

	"github.com/vishvananda/netlink"

	"github.com/zero-os/0-core/base/pm"
)

const (
	ProtocolDHCP = "dhcp"

	carrierFile = "/sys/class/net/%s/carrier"
	waitIPFor   = 30
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
		ID:      fmt.Sprintf("udhcpc/%s", inf),
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

	job, err := pm.Run(cmd)
	if err != nil {
		return err
	}

	link, err := netlink.LinkByName(inf)
	if err != nil {
		return err
	}

	for i := 0; i < waitIPFor; i++ {
		addr, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			return err
		}

		if len(addr) > 0 {
			return nil
		}

		<-time.After(time.Second)
	}

	job.Signal(syscall.SIGTERM)

	return fmt.Errorf("no ip on %s for %d seconds", inf, waitIPFor)
}
