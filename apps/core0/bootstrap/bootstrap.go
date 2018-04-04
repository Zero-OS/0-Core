package bootstrap

import (
	"fmt"
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/op/go-logging"
	"github.com/vishvananda/netlink"
	"github.com/zero-os/0-core/apps/core0/bootstrap/network"
	"github.com/zero-os/0-core/apps/core0/options"
	"github.com/zero-os/0-core/apps/core0/screen"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/settings"
	"github.com/zero-os/0-core/base/utils"
)

const (
	InternetTestAddress = "http://www.google.com/"

	screenStateLine = "->%25s: %s %s"
)

var (
	log = logging.MustGetLogger("bootstrap")
)

type Bootstrap struct {
	i     *settings.IncludedSettings
	t     settings.StartupTree
	agent bool
}

func NewBootstrap(agent bool) *Bootstrap {
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
		i:     included,
		t:     t,
		agent: agent,
	}

	return b
}

func (b *Bootstrap) registerExtensions(extensions map[string]settings.Extension) {
	for extKey, extCfg := range extensions {
		if err := pm.RegisterExtension(extKey, extCfg.Binary, extCfg.Cwd, extCfg.Args, extCfg.Env); err != nil {
			log.Error(err)
		}
	}
}

func (b *Bootstrap) startupServices(s, e settings.After) {
	log.Debugf("Starting up '%s' services", s)
	slice := b.t.Slice(s.Weight(), e.Weight())
	pm.RunSlice(slice)
	log.Debugf("'%s' services are booted", s)
}

func (b *Bootstrap) canReachInternet() bool {
	resp, err := http.Get(InternetTestAddress)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

func (b *Bootstrap) ipsAsString(ips []netlink.Addr) string {
	var s []string
	for _, ip := range ips {
		s = append(s, ip.IPNet.String())
	}

	return strings.Join(s, ", ")
}

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
		inf.SetUP(true)
		go func(inf network.Interface) {
			if err := inf.Configure(); err != nil {
				log.Errorf("%s", err)
			}
		}(inf)
	}

	log.Debugf("waiting for internet reachability")
	now := time.Now()
	for time.Since(now) < time.Minute {
		if b.canReachInternet() {
			log.Info("can reach the internet")
			return nil
		}
	}

	log.Warning("can not reach interent, continue booting anyway")
	return nil
}

func (b *Bootstrap) screen() {
	section := &screen.GroupSection{
		Sections: []screen.Section{},
	}

	screen.Push(section)

	progress := &screen.ProgressSection{}
	reachable := "All Interfaces"
	if options.Options.Kernel.Is("zerotier") {
		reachable = "Zerotier Only"
	}
	reachability := &screen.TextSection{
		Text: fmt.Sprintf(screenStateLine, "Reachability", reachable, ""),
	}

	for {
		links, err := netlink.LinkList()
		if err != nil {
			<-time.After(10 * time.Second)
			continue
		}
		section.Sections = []screen.Section{}

		for _, link := range links {
			if link.Attrs().Name == "lo" || !utils.InString([]string{"device", "tun", "tap"}, link.Type()) {
				continue
			}

			ips, _ := netlink.AddrList(link, netlink.FAMILY_V4)
			section.Sections = append(section.Sections, &screen.TextSection{
				Text: fmt.Sprintf(screenStateLine, link.Attrs().Name, link.Attrs().HardwareAddr, b.ipsAsString(ips)),
			})
		}

		section.Sections = append(section.Sections, progress, reachability)
		screen.Refresh()
		progress.Stop(false)
		progress.Text = fmt.Sprintf(screenStateLine, "Internet Connectivity", "", "")

		if b.canReachInternet() {
			progress.Text = fmt.Sprintf(screenStateLine, "Internet Connectivity", "OK", "")
		} else {
			progress.Text = fmt.Sprintf(screenStateLine, "Internet Connectivity", "NOT OK", "")
		}

		progress.Stop(true)
		screen.Refresh()
		<-time.After(5 * time.Second)
	}
}

func (b *Bootstrap) watchers() {
	screen.Push(&screen.SplitterSection{
		Title: "Zerotier Info",
	})
	section := screen.TextSection{}
	screen.Push(&section)

	go func() {
		for {
			result, err := pm.System("zerotier-cli", "-D/tmp/zt", "info")
			var current string
			if err != nil {
				current = fmt.Sprintf("Cannot show zerotier info due too error: %s",
					strings.TrimSpace(result.Streams.Stderr()),
				)
			}

			current = strings.TrimSpace(result.Streams.Stdout())

			if current != section.Text {
				section.Text = current
				screen.Refresh()
			}

			<-time.After(30 * time.Second)
		}
	}()
}

func (b *Bootstrap) First() {
	if !b.agent {
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{65536, 65536}); err != nil {
			log.Errorf("failed to setup max open files limit: %s", err)
		}

		if err := b.setNFT(); err != nil {
			log.Criticalf("failed to setup NFT: %s", err)
		}
	}

	//register core extensions
	b.registerExtensions(settings.Settings.Extension)

	//register included extensions
	b.registerExtensions(b.i.Extension)
}

//Bootstrap registers extensions and startup system services.
func (b *Bootstrap) Second() {
	progress := &screen.ProgressSection{
		Text: "Bootstrapping: Core Services",
	}
	screen.Push(progress)
	screen.Refresh()

	//start up all init services ([init, net[ slice)
	b.startupServices(settings.AfterInit, settings.AfterNet)

	go b.screen()

	if !b.agent {
		progress.Text = "Bootstrapping: Networking"
		screen.Refresh()
		for {
			err := b.setupNetworking()
			if err == nil {
				break
			}

			log.Errorf("Failed to configure networking: %s", err)
			log.Infof("Retrying in 2 seconds")

			<-time.After(2 * time.Second)
			log.Infof("Retrying setting up network")
		}
	}

	progress.Text = "Bootstrapping: Network Services"
	screen.Refresh()

	//start up all net services ([net, boot[ slice)
	b.startupServices(settings.AfterNet, settings.AfterBoot)

	progress.Text = "Bootstrapping: Services"
	screen.Refresh()

	//start up all boot services ([boot, end] slice)
	b.startupServices(settings.AfterBoot, settings.ToTheEnd)

	progress.Text = "Bootstrapping: Done"
	progress.Stop(true)

	b.watchers()

	screen.Refresh()
}
