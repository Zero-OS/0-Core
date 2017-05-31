package network

import (
	"fmt"
	"github.com/op/go-logging"
	"github.com/vishvananda/netlink"
	"github.com/zero-os/0-core/base/utils"
)

var (
	log = logging.MustGetLogger("network")
)

type interfaceSettings struct {
	Protocol string
}

type staticSettings struct {
	IP      string
	Gateway string
}

type tapSettings struct {
	Up bool
}

type networkSettings struct {
	Auto bool
}

type networkingSettings struct {
	Network   networkSettings
	Interface map[string]interfaceSettings
	Static    map[string]staticSettings
	Tap       map[string]tapSettings
}

type Interface interface {
	Name() string
	Protocol() string
	Configure() error
	Clear() error
	IPs() ([]netlink.Addr, error)
	SetUP(bool) error
}

type NetworkManager interface {
	Initialize() error
	Interfaces() ([]Interface, error)
	getConfig() *networkingSettings
}

type networkInterface struct {
	name     string
	settings interfaceSettings
	manager  NetworkManager
}

func (i *networkInterface) Name() string {
	return i.name
}

func (i *networkInterface) Protocol() string {
	return i.settings.Protocol
}

func (i *networkInterface) SetUP(up bool) error {
	link, err := netlink.LinkByName(i.Name())
	if err != nil {
		return err
	}
	if up {
		return netlink.LinkSetUp(link)
	}

	return netlink.LinkSetDown(link)
}

func (i *networkInterface) Configure() error {
	protocol, ok := protocols[i.Protocol()]
	if !ok {
		return fmt.Errorf("unkonwn networking protocol '%s'", i.Protocol())
	}

	return protocol.Configure(i.manager, i.Name())
}

func (i *networkInterface) Clear() error {
	link, err := netlink.LinkByName(i.Name())
	if err != nil {
		return err
	}
	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		netlink.AddrDel(link, &addr)
	}

	return nil
}

func (i *networkInterface) IPs() ([]netlink.Addr, error) {
	link, err := netlink.LinkByName(i.Name())
	if err != nil {
		return nil, err
	}
	return netlink.AddrList(link, netlink.FAMILY_V4)
}

type networkManager struct {
	settings *networkingSettings
}

//GetNetworkManager gets a new manager and loads settings from the provided toml file
func GetNetworkManager(filename string) (NetworkManager, error) {
	networking := &networkingSettings{}
	if err := utils.LoadTomlFile(filename, networking); err != nil {
		return nil, err
	}

	return &networkManager{networking}, nil
}

/*
getAutoInterfaces gets all interfaces that are NOT manually configured in the configurations
*/
func (m *networkManager) getAutoInterfaces() ([]Interface, error) {
	var l []Interface
	if !m.settings.Network.Auto {
		return l, nil
	}

	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	for _, link := range links {
		if link.Attrs().Name == "lo" {
			continue
		}

		if _, ok := m.settings.Interface[link.Attrs().Name]; ok {
			//if manually configured, just skip
			continue
		}

		inf := &networkInterface{
			name: link.Attrs().Name,
			settings: interfaceSettings{
				Protocol: ProtocolDHCP,
			},
			manager: m,
		}
		l = append(l, inf)
	}

	return l, nil
}

func (m *networkManager) getConfig() *networkingSettings {
	return m.settings
}

/*
Interfaces gets all interfaces as configured by the networking configurations
*/
func (m *networkManager) Interfaces() ([]Interface, error) {
	l, err := m.getAutoInterfaces()
	if err != nil {
		return nil, err
	}

	for k, inf := range m.settings.Interface {
		net := &networkInterface{
			name:     k,
			settings: inf,
			manager:  m,
		}

		l = append(l, net)
	}

	return l, nil
}

func (m *networkManager) Initialize() error {
	for port, cfg := range m.settings.Tap {
		if err := PortCreate(port); err != nil {
			log.Errorf("Failed to create tuntap device '%s': %s", port, err)
		}

		if cfg.Up {
			if err := DeviceUp(port); err != nil {
				log.Errorf("Failed to bring up interface '%s': %s", port, err)
			}
		}
	}
	return nil
}
