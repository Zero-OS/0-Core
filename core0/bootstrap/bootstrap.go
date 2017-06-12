package bootstrap

import (
	"fmt"
	"github.com/op/go-logging"
	"github.com/vishvananda/netlink"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/settings"
	"github.com/zero-os/0-core/base/utils"
	"github.com/zero-os/0-core/core0/bootstrap/network"
	"github.com/zero-os/0-core/core0/options"
	"github.com/zero-os/0-core/core0/screen"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"
)

const (
	InternetTestAddress = "http://www.google.com/"

	screenStateLine = "->%25s: %s %s"

	nft = `
table ip nat {
	chain pre {
		type nat hook prerouting priority 0; policy accept;
	}

	chain post {
		type nat hook postrouting priority 0; policy accept;
	}
}
table ip filter {
	chain input {
		type filter hook input priority 0; policy accept;
	}

	chain forward {
		type filter hook forward priority 0; policy accept;
	}

	chain output {
		type filter hook output priority 0; policy accept;
	}
}
`

	ztOnly = `
table ip filter {
	chain input {
		iifname "zt*" tcp dport 6379 counter packets 0 bytes 0 accept
		tcp dport 6379 counter packets 0 bytes 0 drop
		iifname "zt*" tcp dport 22 counter packets 0 bytes 0 accept
		tcp dport 22 counter packets 0 bytes 0 drop
	}
}
`
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
		if err := inf.Configure(); err != nil {
			log.Errorf("%s", err)
		}
	}

	return nil
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

		section.Sections = append(section.Sections, progress)
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

func (b *Bootstrap) writeRules(r string) (string, error) {
	f, err := ioutil.TempFile("", "nft")
	if err != nil {
		return "", err
	}

	defer f.Close()

	f.WriteString(r)
	return f.Name(), nil
}

func (b *Bootstrap) setNFT() error {

	file, err := b.writeRules(nft)
	if err != nil {
		return err
	}
	defer os.RemoveAll(file)
	if _, err := pm.GetManager().System("nft", "-f", file); err != nil {
		return err
	}

	if options.Options.Kernel.Is("zerotier") && !options.Options.Kernel.Is("debug") {
		file, err := b.writeRules(ztOnly)
		if err != nil {
			return err
		}
		defer os.RemoveAll(file)
		if _, err := pm.GetManager().System("nft", "-f", file); err != nil {
			return err
		}
	}

	return nil
}

//Bootstrap registers extensions and startup system services.
func (b *Bootstrap) Bootstrap() {
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{65536, 65536}); err != nil {
		log.Errorf("failed to setup max open files limit: %s", err)
	}

	if err := b.setNFT(); err != nil {
		log.Criticalf("failed to setup NFT: %s", err)
	}

	//register core extensions
	b.registerExtensions(settings.Settings.Extension)

	//register included extensions
	b.registerExtensions(b.i.Extension)

	progress := &screen.ProgressSection{
		Text: "Bootstrapping: Core Services",
	}
	screen.Push(progress)
	screen.Refresh()

	//start up all init services ([init, net[ slice)
	b.startupServices(settings.AfterInit, settings.AfterNet)

	go b.screen()

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
	screen.Refresh()
}
