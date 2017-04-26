package bootstrap

import (
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/settings"
	"github.com/g8os/core0/base/utils"
	"github.com/g8os/core0/core0/bootstrap/network"
	"github.com/g8os/core0/core0/screen"
	"github.com/op/go-logging"
	"github.com/vishvananda/netlink"
	"net/http"
	"strings"
	"time"
)

const (
	InternetTestAddress = "https://bootstrap.gig.tech/"

	screenStateLine = "->%15s: %s %s"
)

var (
	log = logging.MustGetLogger("bootstrap")
)

type Bootstrap struct {
	i *settings.IncludedSettings
	t settings.StartupTree
}

func NewBootstrap() *Bootstrap {
	included, errors := settings.Settings.GetIncludedSettings()
	if len(errors) > 0 {
		for _, err := range errors {
			log.Errorf("%s", err)
		}
	}

	//startup services from [init, net[
	t, errors := included.GetStartupTree()

	if len(errors) > 0 {
		//print service tree errors (cyclic dependencies, or missing dependencies)
		for _, err := range errors {
			log.Errorf("%s", err)
		}
	}

	b := &Bootstrap{
		i: included,
		t: t,
	}

	return b
}

//TODO: POC bootstrap. This will most probably get rewritten when the process is clearer

func (b *Bootstrap) registerExtensions(extensions map[string]settings.Extension) {
	for extKey, extCfg := range extensions {
		pm.RegisterCmd(extKey, extCfg.Binary, extCfg.Cwd, extCfg.Args, extCfg.Env)
	}
}

func (b *Bootstrap) startupServices(s, e settings.After) {
	log.Debugf("Starting up '%s' services", s)
	slice := b.t.Slice(s.Weight(), e.Weight())
	pm.GetManager().RunSlice(slice)
	log.Debugf("'%s' services are booted", s)
}

func (b *Bootstrap) canReachInternet() bool {
	log.Debugf("Testing internet reachability to %s", InternetTestAddress)
	resp, err := http.Get(InternetTestAddress)
	if err != nil {
		log.Warning("HTTP: %v", err)
		return false
	}
	resp.Body.Close()
	return true
}

//func (b *Bootstrap) initView(infs []network.Interface) *screen.GroupSection {
//	//show interface progress
//	screen.Push(&screen.TextSection{})
//	group := &screen.GroupSection{
//		Sections: map[string]screen.Section{},
//	}
//
//	for _, inf := range infs {
//		group.Sections[inf.Name()] = &screen.ProgressSection{
//			Text: fmt.Sprintf("Setting up %s:", inf.Name()),
//		}
//	}
//	screen.Push(group)
//	screen.Refresh()
//	return group
//}
//
func (b *Bootstrap) ipsAsString(ips []netlink.Addr) string {
	var s []string
	for _, ip := range ips {
		s = append(s, ip.IPNet.String())
	}

	return strings.Join(s, ", ")
}

//
//func (b *Bootstrap) updateView(group *screen.GroupSection, infs []network.Interface) {
//	for _, inf := range infs {
//		ips, err := inf.IPs()
//		if err == nil {
//			progress := group.Sections[inf.Name()]
//			if progress, ok := progress.(*screen.ProgressSection); ok {
//				progress.Stop(true)
//				progress.Text = fmt.Sprintf(screenStateLine, inf.Name(), b.ipsAsString(ips))
//			}
//		}
//	}
//
//	screen.Refresh()
//}

func (b *Bootstrap) setupNetworking() error {
	if settings.Settings.Main.Network == "" {
		log.Warning("No network config file found, skipping network setup")
		return nil
	}

	netMgr, err := network.GetNetworkManager(settings.Settings.Main.Network)
	if err != nil {
		return err
	}

	if err := netMgr.Initialize(); err != nil {
		return err
	}

	interfaces, err := netMgr.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces: %s", err)
	}

	//apply the interfaces settings as configured.
	for _, inf := range interfaces {
		log.Infof("Setting up interface '%s'", inf.Name())

		inf.Clear()
		if err := inf.Configure(); err != nil {
			log.Errorf("%s", err)
		}
	}

	if ok := b.canReachInternet(); ok {
		return nil
	}

	//force dhcp on all interfaces, and try again.
	log.Infof("Trying dhcp on all interfaces one by one")
	dhcp, _ := network.GetProtocol(network.ProtocolDHCP)
	for _, inf := range interfaces {
		//try interfaces one by one
		if inf.Protocol() == network.NoneProtocol || inf.Protocol() == network.ProtocolDHCP || inf.Name() == "lo" {
			//we don't use none interface, they only must be brought up
			//also dhcp interface, we skip because we already tried dhcp method on them.
			//lo device must stay in static.
			continue
		}

		inf.Clear()
		if err := dhcp.Configure(netMgr, inf.Name()); err != nil {
			log.Errorf("Force dhcp %s", err)
		}

		if ok := b.canReachInternet(); ok {
			return nil
		}

		//clear interface
		inf.Clear()
		//reset interface to original setup.
		if err := inf.Configure(); err != nil {
			log.Errorf("%s", err)
		}
	}

	return fmt.Errorf("couldn't reach internet")
}

func (b *Bootstrap) screen() {
	section := &screen.GroupSection{
		Sections: []screen.Section{},
	}

	screen.Push(section)

	progress := &screen.ProgressSection{}
	for {
		links, err := netlink.LinkList()
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		section.Sections = []screen.Section{}

		for _, link := range links {
			if link.Attrs().Name == "lo" || !utils.InString([]string{"device", "tap", "tun"}, link.Type()) {
				continue
			}

			ips, _ := netlink.AddrList(link, netlink.FAMILY_V4)
			section.Sections = append(section.Sections, &screen.TextSection{
				Text: fmt.Sprintf(screenStateLine, link.Attrs().Name, link.Attrs().HardwareAddr, b.ipsAsString(ips)),
			})
		}

		section.Sections = append(section.Sections, progress)
		screen.Refresh()
		progress.Stop(false)
		progress.Text = fmt.Sprintf(screenStateLine, "Connectivity", "", "")

		if b.canReachInternet() {
			progress.Text = fmt.Sprintf(screenStateLine, "Connectivity", "OK", "")
		} else {
			progress.Text = fmt.Sprintf(screenStateLine, "Connectivity", "NOT OK", "")
		}

		progress.Stop(true)
		screen.Refresh()
		time.Sleep(5 * time.Second)
	}
}

//Bootstrap registers extensions and startup system services.
func (b *Bootstrap) Bootstrap() {
	//register core extensions
	b.registerExtensions(settings.Settings.Extension)

	//register included extensions
	b.registerExtensions(b.i.Extension)

	progress := &screen.ProgressSection{
		Text: "Bootstraping: Core Services",
	}
	screen.Push(progress)
	screen.Refresh()

	//start up all init services ([init, net[ slice)
	b.startupServices(settings.AfterInit, settings.AfterNet)

	go b.screen()

	progress.Text = "Bootstraping: Networking"
	screen.Refresh()
	for {
		err := b.setupNetworking()
		if err == nil {
			break
		}

		log.Errorf("Failed to configure networking: %s", err)
		log.Infof("Retrying in 2 seconds")

		time.Sleep(2 * time.Second)
		log.Infof("Retrying setting up network")
	}

	progress.Text = "Bootstraping: Network Services"
	screen.Refresh()

	//start up all net services ([net, boot[ slice)
	b.startupServices(settings.AfterNet, settings.AfterBoot)

	progress.Text = "Bootstraping: Services"
	screen.Refresh()

	//start up all boot services ([boot, end] slice)
	b.startupServices(settings.AfterBoot, settings.ToTheEnd)

	progress.Text = "Bootstraping: Done"
	progress.Stop(true)
	screen.Refresh()
}
