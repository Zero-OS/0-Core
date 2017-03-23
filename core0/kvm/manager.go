package kvm

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/url"
	//"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/pborman/uuid"
	"github.com/vishvananda/netlink"
	"sync"
)

const (
	BaseMACAddress = "00:28:06:82:%x:%x"

	BaseIPAddr = "172.19.%d.%d"
)

type kvmManager struct {
	sequence uint16
	m        sync.Mutex
}

var (
	pattern = regexp.MustCompile(`^\s*(\d+)(.+)\s(\w+)$`)

	ReservedSequences = []uint16{0x0, 0x1, 0xffff}
	BridgeIP          = []byte{172, 19, 0, 1}
	IPRangeStart      = fmt.Sprintf("%d.%d.%d.%d", BridgeIP[0], BridgeIP[1], 0, 2)
	IPRangeEnd        = fmt.Sprintf("%d.%d.%d.%d", BridgeIP[0], BridgeIP[1], 255, 254)
	DefaultBridgeIP   = fmt.Sprintf("%d.%d.%d.%d", BridgeIP[0], BridgeIP[1], BridgeIP[2], BridgeIP[3])
	DefaultBridgeCIDR = fmt.Sprintf("%s/16", DefaultBridgeIP)
)

const (
	kvmCreateCommand  = "kvm.create"
	kvmDestroyCommand = "kvm.destroy"
	kvmListCommand    = "kvm.list"

	DefaultBridgeName = "kvm-0"
)

func KVMSubsystem() error {
	mgr := &kvmManager{}

	if err := mgr.setupDefaultGateway(); err != nil {
		return err
	}

	pm.CmdMap[kvmCreateCommand] = process.NewInternalProcessFactory(mgr.create)
	pm.CmdMap[kvmDestroyCommand] = process.NewInternalProcessFactory(mgr.destroy)
	pm.CmdMap[kvmListCommand] = process.NewInternalProcessFactory(mgr.list)

	return nil
}

type Media struct {
	URL  string         `json:"url"`
	Type DiskDeviceType `json:"type"`
	Bus  string         `json:"bus"`
}

type CreateParams struct {
	Name   string      `json:"name"`
	CPU    int         `json:"cpu"`
	Memory int         `json:"memory"`
	Media  []Media     `json:"media"`
	Bridge []string    `json:"bridge"`
	Port   map[int]int `json:"port"`
}

func (m *kvmManager) setupDefaultGateway() error {
	cmd := &core.Command{
		ID:      uuid.New(),
		Command: "bridge.create",
		Arguments: core.MustArguments(
			core.M{
				"name": DefaultBridgeName,
				"network": core.M{
					"nat":  true,
					"mode": "dnsmasq",
					"settings": core.M{
						"cidr":  DefaultBridgeCIDR,
						"start": IPRangeStart,
						"end":   IPRangeEnd,
					},
				},
			},
		),
	}

	runner, err := pm.GetManager().RunCmd(cmd)
	if err != nil {
		return err
	}
	result := runner.Wait()
	if result.State != core.StateSuccess {
		return fmt.Errorf("failed to create default container bridge: %s", result.Data)
	}

	return nil
}

func (m *kvmManager) mkNBDDisk(idx int, u *url.URL) DiskDevice {
	name := strings.Trim(u.Path, "/")
	target := "vd" + string(97+idx)

	switch u.Scheme {
	case "nbd":
		fallthrough
	case "nbd+tcp":
		port := u.Port()
		if port == "" {
			port = "10809"
		}
		return DiskDevice{
			Type: DiskTypeNetwork,
			Target: DiskTarget{
				Dev: target,
			},
			Source: DiskSourceNetwork{
				Protocol: "nbd",
				Name:     name,
				Host: DiskSourceNetworkHost{
					Transport: "tcp",
					Port:      port,
					Name:      u.Hostname(),
				},
			},
		}
	case "nbd+unix":
		return DiskDevice{
			Type: DiskTypeNetwork,
			Target: DiskTarget{
				Dev: target,
			},
			Source: DiskSourceNetwork{
				Protocol: "nbd",
				Name:     name,
				Host: DiskSourceNetworkHost{
					Transport: "unix",
					Socket:    u.Query().Get("socket"),
				},
			},
		}
	default:
		panic(fmt.Errorf("invalid nbd url: %s", u))
	}
}

func (m *kvmManager) mkFileDisk(idx int, u *url.URL) DiskDevice {
	target := "vd" + string(97+idx)
	return DiskDevice{
		Type: DiskTypeFile,
		Target: DiskTarget{
			Dev: target,
		},
		Source: DiskSourceFile{
			File: u.String(),
		},
	}
}

func (m *kvmManager) mkDisk(idx int, media Media) DiskDevice {
	u, err := url.Parse(media.URL)

	var disk DiskDevice
	if err == nil && strings.Index(u.Scheme, "nbd") == 0 {
		disk = m.mkNBDDisk(idx, u)
	} else {
		disk = m.mkFileDisk(idx, u)
	}

	disk.Device = DiskDeviceTypeDisk
	if media.Type != DiskDeviceType("") {
		disk.Device = media.Type
	}

	disk.Target.Bus = "virtio"
	if media.Bus != "" {
		disk.Target.Bus = media.Bus
	}

	//hack for cdrom, because it doesn't work well with virtio
	if media.Type == DiskDeviceTypeCDROM {
		disk.Target.Dev = "hd" + string(97+idx)
		disk.Target.Bus = "ide"
	}

	return disk
}

func (m *kvmManager) getNextSequence() uint16 {
	m.m.Lock()
	defer m.m.Unlock()
loop:
	for {
		m.sequence += 1
		for _, r := range ReservedSequences {
			if m.sequence == r {
				continue loop
			}
		}
		break
	}

	return m.sequence
}

func (m *kvmManager) macAddr(s uint16) string {
	return fmt.Sprintf(BaseMACAddress,
		(s & 0x0000FF00 >> 8),
		(s & 0x000000FF),
	)
}

func (m *kvmManager) ipAddr(s uint16) string {
	return fmt.Sprintf("%d.%d.%d.%d", BridgeIP[0], BridgeIP[1], (s&0xff00)>>8, s&0x00ff)
}

func (m *kvmManager) mkDomain(seq uint16, params *CreateParams) (*Domain, error) {

	domain := Domain{
		Type: DomainTypeKVM,
		Name: params.Name,
		UUID: uuid.New(),
		Memory: Memory{
			Capacity: params.Memory,
			Unit:     "MB",
		},
		VCPU: params.CPU,
		OS: OS{
			Type: OSType{
				Type: OSTypeTypeHVM,
				Arch: ArchX86_64,
			},
		},
		Devices: Devices{
			Emulator: "/usr/bin/qemu-system-x86_64",
			Devices: []Device{
				SerialDevice{
					Type: SerialDeviceTypePTY,
					Source: SerialSource{
						Path: "/dev/pts/1",
					},
					Target: SerialTarget{
						Port: 0,
					},
					Alias: SerialAlias{
						Name: "serial0",
					},
				},
				ConsoleDevice{
					Type: SerialDeviceTypePTY,
					TTY:  "/dev/pts/1",
					Source: SerialSource{
						Path: "/dev/pts/1",
					},
					Target: ConsoleTarget{
						Port: 0,
						Type: "serial",
					},
					Alias: SerialAlias{
						Name: "serial0",
					},
				},
				GraphicsDevice{
					Type:   GraphicsDeviceTypeVNC,
					Port:   -1,
					KeyMap: "en-us",
					Listen: Listen{
						Type:    "address",
						Address: "0.0.0.0",
					},
				},
			},
		},
	}

	//attach to default bridge.
	domain.Devices.Devices = append(domain.Devices.Devices, InterfaceDevice{
		Type: InterfaceDeviceTypeBridge,
		Source: InterfaceDeviceSourceBridge{
			Bridge: DefaultBridgeName,
		},
		Mac: &InterfaceDeviceMac{
			Address: m.macAddr(seq),
		},
		Model: InterfaceDeviceModel{
			Type: "virtio",
		},
	})

	for _, bridge := range params.Bridge {
		_, err := netlink.LinkByName(bridge)
		if err != nil {
			return nil, fmt.Errorf("bridge '%s' not found", bridge)
		}

		domain.Devices.Devices = append(domain.Devices.Devices, InterfaceDevice{
			Type: InterfaceDeviceTypeBridge,
			Source: InterfaceDeviceSourceBridge{
				Bridge: bridge,
			},
			Model: InterfaceDeviceModel{
				Type: "virtio",
			},
		})
	}

	for idx, media := range params.Media {
		domain.Devices.Devices = append(domain.Devices.Devices, m.mkDisk(idx, media))
	}

	return &domain, nil
}

func (m *kvmManager) configureDhcpHost(seq uint16) error {
	mac := m.macAddr(seq)
	ip := m.ipAddr(seq)

	runner, err := pm.GetManager().RunCmd(&core.Command{
		ID:      uuid.New(),
		Command: "bridge.add_host",
		Arguments: core.MustArguments(map[string]interface{}{
			"bridge": DefaultBridgeName,
			"mac":    mac,
			"ip":     ip,
		}),
	})

	if err != nil {
		return err
	}
	result := runner.Wait()

	if result.State != core.StateSuccess {
		return fmt.Errorf("failed to add host to dnsmasq: %s", result.Data)
	}

	return nil
}

func (m *kvmManager) forwardId(name string, host int) string {
	return fmt.Sprintf("kvm-socat-%s-%d", name, host)
}

func (m *kvmManager) unPortForward(name string) {
	for key, runner := range pm.GetManager().Runners() {
		if strings.HasPrefix(key, fmt.Sprintf("kvm-socat-%s", name)) {
			runner.Kill()
		}
	}
}

func (m *kvmManager) setPortForwards(seq uint16, params *CreateParams) error {
	ip := m.ipAddr(seq)

	for host, container := range params.Port {
		//nft add rule nat prerouting iif eth0 tcp dport { 80, 443 } dnat 192.168.1.120
		cmd := &core.Command{
			ID:      m.forwardId(params.Name, host),
			Command: process.CommandSystem,
			Arguments: core.MustArguments(
				process.SystemCommandArguments{
					Name: "socat",
					Args: []string{
						fmt.Sprintf("tcp-listen:%d,reuseaddr,fork", host),
						fmt.Sprintf("tcp-connect:%s:%d", ip, container),
					},
					NoOutput: true,
				},
			),
		}

		pm.GetManager().RunCmd(cmd)
	}

	return nil
}

func (m *kvmManager) create(cmd *core.Command) (interface{}, error) {
	var params CreateParams
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}

	seq := m.getNextSequence()

	domain, err := m.mkDomain(seq, &params)
	if err != nil {
		return nil, err
	}

	if err := m.configureDhcpHost(seq); err != nil {
		return nil, err
	}

	data, err := xml.MarshalIndent(domain, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to generate domain xml: %s", err)
	}

	tmp, err := ioutil.TempFile("/tmp", "kvm-domain")
	if err != nil {
		return nil, err
	}
	//defer os.Remove(tmp.Name())
	defer tmp.Close()

	if _, err := tmp.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write domain xml: %s", err)
	}

	tmp.Close()

	//create domain
	virsh := &core.Command{
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: "virsh",
				Args: []string{
					"create", tmp.Name(),
				},
			},
		),
	}
	runner, err := pm.GetManager().RunCmd(virsh)
	if err != nil {
		return nil, fmt.Errorf("failed to start virsh: %s", err)
	}
	result := runner.Wait()
	if result.State != core.StateSuccess {
		return nil, fmt.Errorf(result.Streams[1])
	}

	//start port forwarders
	if err := m.setPortForwards(seq, &params); err != nil {
		return nil, err
	}
	return nil, nil
}

type DestroyParams struct {
	Name string `json:"name"`
}

func (m *kvmManager) destroy(cmd *core.Command) (interface{}, error) {
	var params DestroyParams
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}
	virsh := &core.Command{
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: "virsh",
				Args: []string{
					"destroy", params.Name,
				},
			},
		),
	}
	runner, err := pm.GetManager().RunCmd(virsh)
	if err != nil {
		return nil, fmt.Errorf("failed to destroy machine: %s", err)
	}
	result := runner.Wait()
	if result.State != core.StateSuccess {
		return nil, fmt.Errorf(result.Streams[1])
	}

	m.unPortForward(params.Name)

	return nil, nil
}

type Machine struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

func (m *kvmManager) list(cmd *core.Command) (interface{}, error) {
	virsh := &core.Command{
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: "virsh",
				Args: []string{
					"list", "--all",
				},
			},
		),
	}
	runner, err := pm.GetManager().RunCmd(virsh)
	if err != nil {
		return nil, fmt.Errorf("failed to destroy machine: %s", err)
	}
	result := runner.Wait()
	if result.State != core.StateSuccess {
		return nil, fmt.Errorf(result.Streams[1])
	}

	out := result.Streams[0]

	found := make([]Machine, 0)
	lines := strings.Split(out, "\n")
	if len(lines) <= 3 {
		return found, nil
	}

	lines = lines[2:]

	for _, line := range lines {
		match := pattern.FindStringSubmatch(line)
		if len(match) != 4 {
			continue
		}
		id, _ := strconv.ParseInt(match[1], 10, 32)
		found = append(found, Machine{
			ID:    int(id),
			Name:  strings.TrimSpace(match[2]),
			State: strings.TrimSpace(match[3]),
		})
	}

	return found, nil
}
