package bootstrap

import (
	"fmt"
	"github.com/g8os/core.base/pm"
	"github.com/g8os/core.base/settings"
	"github.com/g8os/core0/network"
	"github.com/op/go-logging"
	"net/http"
	"time"
)

const (
	InternetTestAddress = "http://www.google.com"
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
	log.Infof("Starting up '%s' services", s)
	slice := b.t.Slice(s.Weight(), e.Weight())
	pm.GetManager().RunSlice(slice)
}

func (b *Bootstrap) canReachInternet() bool {
	log.Infof("Testing internet reachability to %s", InternetTestAddress)
	resp, err := http.Get(InternetTestAddress)
	if err != nil {
		log.Warning("HTTP: %v", err)
		return false
	}
	resp.Body.Close()
	return true
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
		if err := inf.Configure(); err != nil {
			log.Errorf("%s", err)
		}
	}

	if ok := b.canReachInternet(); ok {
		return nil
	}

	//force dhcp on all interfaces, and try again.
	log.Infof("Trying dhcp on all interfaces one by one")
	proto, _ := network.GetProtocol(network.ProtocolDHCP)
	dhcp := proto.(network.DHCPProtocol)
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
		//stop dhcp
		dhcp.Stop(inf.Name())
		//clear interface
		inf.Clear()
		//reset interface to original setup.
		if err := inf.Configure(); err != nil {
			log.Errorf("%s", err)
		}
	}

	return fmt.Errorf("couldn't reach internet")
}

//Bootstrap registers extensions and startup system services.
func (b *Bootstrap) Bootstrap() {
	//register core extensions
	b.registerExtensions(settings.Settings.Extension)

	//register included extensions
	b.registerExtensions(b.i.Extension)

	//start up all init services ([init, net[ slice)
	b.startupServices(settings.AfterInit, settings.AfterNet)

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

	//start up all net services ([net, boot[ slice)
	b.startupServices(settings.AfterNet, settings.AfterBoot)

	//start up all boot services ([boot, end] slice)
	b.startupServices(settings.AfterBoot, settings.ToTheEnd)
}
