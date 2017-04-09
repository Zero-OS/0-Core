package kvm

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	//"os"
	"regexp"
	"strings"

	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/libvirt/libvirt-go"
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
	kvmCreateCommand      = "kvm.create"
	kvmDestroyCommand     = "kvm.destroy"
	kvmShutdownCommand    = "kvm.shutdown"
	kvmRebootCommand      = "kvm.reboot"
	kvmResetCommand       = "kvm.reset"
	kvmPauseCommand       = "kvm.pause"
	kvmResumeCommand      = "kvm.resume"
	kvmAttachDiskCommand  = "kvm.attachDisk"
	kvmDetachDiskCommand  = "kvm.detachDisk"
	kvmAddNicCommand      = "kvm.addNic"
	kvmRemoveNicCommand   = "kvm.removeNic"
	kvmLimitDiskIOCommand = "kvm.limitDiskIO"
	kvmMigrateCommand     = "kvm.migrate"
	kvmListCommand        = "kvm.list"

	DefaultBridgeName = "kvm-0"
)

func KVMSubsystem() error {
	mgr := &kvmManager{}

	if err := mgr.setupDefaultGateway(); err != nil {
		return err
	}

	pm.CmdMap[kvmCreateCommand] = process.NewInternalProcessFactory(mgr.create)
	pm.CmdMap[kvmDestroyCommand] = process.NewInternalProcessFactory(mgr.destroy)
	pm.CmdMap[kvmShutdownCommand] = process.NewInternalProcessFactory(mgr.shutdown)
	pm.CmdMap[kvmRebootCommand] = process.NewInternalProcessFactory(mgr.reboot)
	pm.CmdMap[kvmResetCommand] = process.NewInternalProcessFactory(mgr.reset)
	pm.CmdMap[kvmPauseCommand] = process.NewInternalProcessFactory(mgr.pause)
	pm.CmdMap[kvmResumeCommand] = process.NewInternalProcessFactory(mgr.resume)
	pm.CmdMap[kvmAttachDiskCommand] = process.NewInternalProcessFactory(mgr.attachDisk)
	pm.CmdMap[kvmDetachDiskCommand] = process.NewInternalProcessFactory(mgr.detachDisk)
	pm.CmdMap[kvmAddNicCommand] = process.NewInternalProcessFactory(mgr.addNic)
	pm.CmdMap[kvmRemoveNicCommand] = process.NewInternalProcessFactory(mgr.removeNic)
	pm.CmdMap[kvmLimitDiskIOCommand] = process.NewInternalProcessFactory(mgr.limitDiskIO)
	pm.CmdMap[kvmMigrateCommand] = process.NewInternalProcessFactory(mgr.migrate)
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

type DomainUUID struct {
	UUID string `json:"uuid"`
}

type ManDiskParams struct {
	UUID  string `json:"uuid"`
	Media Media  `json:"media"`
}

type ManNicParams struct {
	UUID   string `json:"uuid"`
	Bridge string `json:"bridge"`
}

type MigrateParams struct {
	UUID    string `json:"uuid"`
	DestURI string `json:"desturi"`
}

type LimitDiskIOParams struct {
	UUID                      string `json:"uuid"`
	TargetName                string `json:"targetname"`
	TotalBytesSecSet          bool   `json:"totalbytessecset"`
	TotalBytesSec             uint64 `json:"totalbytessec"`
	ReadBytesSecSet           bool   `json:"readbytessecset"`
	ReadBytesSec              uint64 `json:"readbytessec"`
	WriteBytesSecSet          bool   `json:"writebytessecset"`
	WriteBytesSec             uint64 `json:"writebytessec"`
	TotalIopsSecSet           bool   `json:"totaliopssecset"`
	TotalIopsSec              uint64 `json:"totaliopssec"`
	ReadIopsSecSet            bool   `json:"readiopssecset"`
	ReadIopsSec               uint64 `json:"readiopssec"`
	WriteIopsSecSet           bool   `json:"writeiopssecset"`
	WriteIopsSec              uint64 `json:"writeiopssec"`
	TotalBytesSecMaxSet       bool   `json:"totalbytessecmaxset"`
	TotalBytesSecMax          uint64 `json:"totalbytessecmax"`
	ReadBytesSecMaxSet        bool   `json:"readbytessecmaxset"`
	ReadBytesSecMax           uint64 `json:"readbytessecmax"`
	WriteBytesSecMaxSet       bool   `json:"writebytessecmaxset"`
	WriteBytesSecMax          uint64 `json:"writebytessecmax"`
	TotalIopsSecMaxSet        bool   `json:"totaliopssecmaxset"`
	TotalIopsSecMax           uint64 `json:"totaliopssecmax"`
	ReadIopsSecMaxSet         bool   `json:"readiopssecmaxset"`
	ReadIopsSecMax            uint64 `json:"readiopssecmax"`
	WriteIopsSecMaxSet        bool   `json:"writeiopssecmaxset"`
	WriteIopsSecMax           uint64 `json:"writeiopssecmax"`
	TotalBytesSecMaxLengthSet bool   `json:"totalbytessecmaxlengthset"`
	TotalBytesSecMaxLength    uint64 `json:"totalbytessecmaxlength"`
	ReadBytesSecMaxLengthSet  bool   `json:"readbytessecmaxlengthset"`
	ReadBytesSecMaxLength     uint64 `json:"readbytessecmaxlength"`
	WriteBytesSecMaxLengthSet bool   `json:"writebytessecmaxlengthset"`
	WriteBytesSecMaxLength    uint64 `json:"writebytessecmaxlength"`
	TotalIopsSecMaxLengthSet  bool   `json:"totaliopssecmaxlengthset"`
	TotalIopsSecMaxLength     uint64 `json:"totaliopssecmaxlength"`
	ReadIopsSecMaxLengthSet   bool   `json:"readiopssecmaxlengthset"`
	ReadIopsSecMaxLength      uint64 `json:"readiopssecmaxlength"`
	WriteIopsSecMaxLengthSet  bool   `json:"writeiopssecmaxlengthset"`
	WriteIopsSecMaxLength     uint64 `json:"writeiopssecmaxlength"`
	SizeIopsSecSet            bool   `json:"sizeiopssecset"`
	SizeIopsSec               uint64 `json:"sizeiopssec"`
	GroupNameSet              bool   `json:"groupnameset"`
	GroupName                 string `json:"groupname"`
}

func StateToString(state libvirt.DomainState) string {
	var res string
	switch state {
	case libvirt.DOMAIN_NOSTATE:
		res = "nostate"
	case libvirt.DOMAIN_RUNNING:
		res = "running"
	case libvirt.DOMAIN_BLOCKED:
		res = "blocked"
	case libvirt.DOMAIN_PAUSED:
		res = "paused"
	case libvirt.DOMAIN_SHUTDOWN:
		res = "shutdown"
	case libvirt.DOMAIN_CRASHED:
		res = "crashed"
	case libvirt.DOMAIN_PMSUSPENDED:
		res = "pmsuspended"
	case libvirt.DOMAIN_SHUTOFF:
		res = "shutoff"
	default:
		res = ""
	}
	return res
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
			Emulator:   "/usr/bin/qemu-system-x86_64",
			Disks:      []DiskDevice{},
			Interfaces: []InterfaceDevice{},
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

		domain.Devices.Interfaces = append(domain.Devices.Interfaces, InterfaceDevice{
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
		domain.Devices.Disks = append(domain.Devices.Disks, m.mkDisk(idx, media))
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

func (m *kvmManager) forwardId(uuid string, host int) string {
	return fmt.Sprintf("kvm-socat-%s-%d", uuid, host)
}

func (m *kvmManager) unPortForward(uuid string) {
	for key, runner := range pm.GetManager().Runners() {
		if strings.HasPrefix(key, fmt.Sprintf("kvm-socat-%s", uuid)) {
			runner.Terminate()
		}
	}
}

func (m *kvmManager) setPortForwards(uuid string, seq uint16, port map[int]int) error {
	ip := m.ipAddr(seq)

	for host, container := range port {
		//nft add rule nat prerouting iif eth0 tcp dport { 80, 443 } dnat 192.168.1.120
		cmd := &core.Command{
			ID:      m.forwardId(uuid, host),
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

	//create domain
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return nil, fmt.Errorf("failed to start a qemu connection: %s", err)
	}
	defer conn.Close()

	dom, err := conn.DomainCreateXML(string(data[:]), libvirt.DOMAIN_NONE)
	if err != nil {
		return nil, fmt.Errorf("failed to create machine: %s", err)
	}

	uuid, err := dom.GetUUIDString()
	if err != nil {
		return nil, fmt.Errorf("failed to get machine uuid with the name %s", params.Name)
	}

	//start port forwarders
	if err := m.setPortForwards(uuid, seq, params.Port); err != nil {
		return nil, err
	}

	return DomainUUID{uuid}, nil
}

func (m *kvmManager) getDomain(cmd *core.Command) (*libvirt.Domain, *libvirt.Connect, string, error) {
	var params DomainUUID
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, nil, "", err
	}

	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return nil, nil, params.UUID, fmt.Errorf("failed to start a qemu connection: %s", err)
	}

	domain, err := conn.LookupDomainByUUIDString(params.UUID)
	if err != nil {
		conn.Close()
		return nil, nil, params.UUID, fmt.Errorf("couldn't find domain with the uuid %s", params.UUID)
	}
	// we don't close the connection here because it is supposed to be used outside
	// so we expect the caller to close it
	// so if anything is to be added in this method that can return an error
	// the connection has to be closed before the return
	return domain, conn, params.UUID, err
}

func (m *kvmManager) destroy(cmd *core.Command) (interface{}, error) {
	domain, conn, uuid, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := domain.Destroy(); err != nil {
		return nil, fmt.Errorf("failed to destroy machine: %s", err)
	}
	m.unPortForward(uuid)

	return nil, nil
}

func (m *kvmManager) shutdown(cmd *core.Command) (interface{}, error) {
	domain, conn, uuid, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := domain.Shutdown(); err != nil {
		return nil, fmt.Errorf("failed to shutdown machine: %s", err)
	}

	m.unPortForward(uuid)

	return nil, nil
}

func (m *kvmManager) reboot(cmd *core.Command) (interface{}, error) {
	domain, conn, _, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := domain.Reboot(libvirt.DOMAIN_REBOOT_DEFAULT); err != nil {
		return nil, fmt.Errorf("failed to reboot machine: %s", err)
	}

	return nil, nil
}

func (m *kvmManager) reset(cmd *core.Command) (interface{}, error) {
	domain, conn, _, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := domain.Reset(0); err != nil {
		return nil, fmt.Errorf("failed to reset machine: %s", err)
	}

	return nil, nil
}

func (m *kvmManager) pause(cmd *core.Command) (interface{}, error) {
	domain, conn, _, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := domain.Suspend(); err != nil {
		return nil, fmt.Errorf("failed to pause machine: %s", err)
	}

	return nil, nil
}

func (m *kvmManager) resume(cmd *core.Command) (interface{}, error) {
	domain, conn, _, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := domain.Resume(); err != nil {
		return nil, fmt.Errorf("failed to resume machine: %s", err)
	}

	return nil, nil
}

func (m *kvmManager) attachDevice(uuid, xml string) error {
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return fmt.Errorf("failed to start a qemu connection: %s", err)
	}
	defer conn.Close()

	domain, err := conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return fmt.Errorf("couldn't find domain with the uuid %s", uuid)
	}
	if err := domain.AttachDeviceFlags(xml, libvirt.DOMAIN_DEVICE_MODIFY_LIVE); err != nil {
		return fmt.Errorf("failed to attach device: %s", err)
	}

	return nil
}

func (m *kvmManager) detachDevice(uuid, xml string) error {
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return fmt.Errorf("failed to start a qemu connection: %s", err)
	}
	defer conn.Close()

	domain, err := conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return fmt.Errorf("couldn't find domain with the uuid %s", uuid)
	}
	if err := domain.DetachDeviceFlags(xml, libvirt.DOMAIN_DEVICE_MODIFY_LIVE); err != nil {
		return fmt.Errorf("failed to attach device: %s", err)
	}

	return nil
}

func (m *kvmManager) attachDisk(cmd *core.Command) (interface{}, error) {
	var params ManDiskParams
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return nil, fmt.Errorf("failed to start a qemu connection: %s", err)
	}
	defer conn.Close()

	domain, err := conn.LookupDomainByUUIDString(params.UUID)
	if err != nil {
		return nil, fmt.Errorf("couldn't find domain with the uuid %s", params.UUID)
	}
	domainxml, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return nil, fmt.Errorf("cannot get domain xml: %v", err)
	}
	domainstruct := Domain{}
	err = xml.Unmarshal([]byte(domainxml), &domainstruct)
	if err != nil {
		return nil, fmt.Errorf("cannot parse the domain xml: %v", err)
	}
	count := len(domainstruct.Devices.Disks)
	disk := m.mkDisk(count, params.Media)
	diskxml, err := xml.MarshalIndent(disk, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot marshal disk to xml")
	}
	return nil, m.attachDevice(params.UUID, string(diskxml[:]))
}

func (m *kvmManager) detachDisk(cmd *core.Command) (interface{}, error) {
	var params ManDiskParams
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}
	// FIXME: get the idx of the disk
	idx := 0
	media := params.Media
	disk := m.mkDisk(idx, media)
	diskxml, err := xml.MarshalIndent(disk, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot marshal disk to xml")
	}
	return nil, m.detachDevice(params.UUID, string(diskxml[:]))
}

func (m *kvmManager) addNic(cmd *core.Command) (interface{}, error) {
	var params ManNicParams
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}
	bridge := params.Bridge
	_, err := netlink.LinkByName(bridge)
	if err != nil {
		return nil, fmt.Errorf("bridge '%s' not found", bridge)
	}

	ifd := InterfaceDevice{
		Type: InterfaceDeviceTypeBridge,
		Source: InterfaceDeviceSourceBridge{
			Bridge: bridge,
		},
		Model: InterfaceDeviceModel{
			Type: "virtio",
		},
	}
	ifxml, err := xml.MarshalIndent(ifd, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot marshal nic to xml")
	}
	return nil, m.attachDevice(params.UUID, string(ifxml[:]))
}

func (m *kvmManager) removeNic(cmd *core.Command) (interface{}, error) {
	var params ManNicParams
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}
	bridge := params.Bridge
	ifd := InterfaceDevice{
		Type: InterfaceDeviceTypeBridge,
		Source: InterfaceDeviceSourceBridge{
			Bridge: bridge,
		},
		Model: InterfaceDeviceModel{
			Type: "virtio",
		},
	}
	ifxml, err := xml.MarshalIndent(ifd, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot marshal nic to xml")
	}
	return nil, m.detachDevice(params.UUID, string(ifxml[:]))
}

func (m *kvmManager) limitDiskIO(cmd *core.Command) (interface{}, error) {
	var params LimitDiskIOParams
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return nil, fmt.Errorf("failed to start a qemu connection: %s", err)
	}
	defer conn.Close()

	domain, err := conn.LookupDomainByUUIDString(params.UUID)
	if err != nil {
		return nil, fmt.Errorf("couldn't find domain with the uuid %s", params.UUID)
	}
	blockParams := libvirt.DomainBlockIoTuneParameters{
		TotalBytesSecSet:          params.TotalBytesSecSet,
		TotalBytesSec:             params.TotalBytesSec,
		ReadBytesSecSet:           params.ReadBytesSecSet,
		ReadBytesSec:              params.ReadBytesSec,
		WriteBytesSecSet:          params.WriteBytesSecSet,
		WriteBytesSec:             params.WriteBytesSec,
		TotalIopsSecSet:           params.TotalIopsSecSet,
		TotalIopsSec:              params.TotalIopsSec,
		ReadIopsSecSet:            params.ReadIopsSecSet,
		ReadIopsSec:               params.ReadIopsSec,
		WriteIopsSecSet:           params.WriteIopsSecSet,
		WriteIopsSec:              params.WriteIopsSec,
		TotalBytesSecMaxSet:       params.TotalBytesSecMaxSet,
		TotalBytesSecMax:          params.TotalBytesSecMax,
		ReadBytesSecMaxSet:        params.ReadBytesSecMaxSet,
		ReadBytesSecMax:           params.ReadBytesSecMax,
		WriteBytesSecMaxSet:       params.WriteBytesSecMaxSet,
		WriteBytesSecMax:          params.WriteBytesSecMax,
		TotalIopsSecMaxSet:        params.TotalIopsSecMaxSet,
		TotalIopsSecMax:           params.TotalIopsSecMax,
		ReadIopsSecMaxSet:         params.ReadIopsSecMaxSet,
		ReadIopsSecMax:            params.ReadIopsSecMax,
		WriteIopsSecMaxSet:        params.WriteIopsSecMaxSet,
		WriteIopsSecMax:           params.WriteIopsSecMax,
		TotalBytesSecMaxLengthSet: params.TotalBytesSecMaxLengthSet,
		TotalBytesSecMaxLength:    params.TotalBytesSecMaxLength,
		ReadBytesSecMaxLengthSet:  params.ReadBytesSecMaxLengthSet,
		ReadBytesSecMaxLength:     params.ReadBytesSecMaxLength,
		WriteBytesSecMaxLengthSet: params.WriteBytesSecMaxLengthSet,
		WriteBytesSecMaxLength:    params.WriteBytesSecMaxLength,
		TotalIopsSecMaxLengthSet:  params.TotalIopsSecMaxLengthSet,
		TotalIopsSecMaxLength:     params.TotalIopsSecMaxLength,
		ReadIopsSecMaxLengthSet:   params.ReadIopsSecMaxLengthSet,
		ReadIopsSecMaxLength:      params.ReadIopsSecMaxLength,
		WriteIopsSecMaxLengthSet:  params.WriteIopsSecMaxLengthSet,
		WriteIopsSecMaxLength:     params.WriteIopsSecMaxLength,
		SizeIopsSecSet:            params.SizeIopsSecSet,
		SizeIopsSec:               params.SizeIopsSec,
		GroupNameSet:              params.GroupNameSet,
		GroupName:                 params.GroupName,
	}
	if err := domain.SetBlockIoTune(params.TargetName, &blockParams, libvirt.DOMAIN_AFFECT_LIVE); err != nil {
		return nil, fmt.Errorf("failed to tune disk: %s", err)
	}
	return nil, nil
}

func (m *kvmManager) migrate(cmd *core.Command) (interface{}, error) {
	domain, conn, _, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	var params MigrateParams
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}
	dconn, err := libvirt.NewConnect(params.DestURI)
	if err != nil {
		return nil, fmt.Errorf("failed to start a qemu connection: %s", err)
	}
	defer dconn.Close()
	dxml, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE)
	if err != nil {
		return nil, err
	}
	name, err := domain.GetName()
	if err != nil {
		return nil, err
	}
	if _, err = domain.Migrate2(dconn, dxml, libvirt.MIGRATE_LIVE|libvirt.MIGRATE_UNDEFINE_SOURCE, name, "", 10000000000); err != nil {
		return nil, err
	}
	return nil, nil
}

type Machine struct {
	ID    int    `json:"id"`
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
	State string `json:"state"`
}

func (m *kvmManager) list(cmd *core.Command) (interface{}, error) {
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return nil, fmt.Errorf("failed to start a qemu connection: %s", err)
	}
	defer conn.Close()

	domains, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return nil, fmt.Errorf("failed to list machines: %s", err)
	}

	found := make([]Machine, 0)

	for _, domain := range domains {
		id, err := domain.GetID()
		if err != nil {
			return nil, err
		}
		uuid, err := domain.GetUUIDString()
		if err != nil {
			return nil, err
		}
		name, err := domain.GetName()
		if err != nil {
			return nil, err
		}
		state, _, err := domain.GetState()
		if err != nil {
			return nil, err
		}
		found = append(found, Machine{
			ID:    int(id),
			UUID:  uuid,
			Name:  name,
			State: StateToString(state),
		})
	}

	return found, nil
}
