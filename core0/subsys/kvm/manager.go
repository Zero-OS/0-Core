package kvm

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/libvirt/libvirt-go"
	"github.com/op/go-logging"
	"github.com/pborman/uuid"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"github.com/zero-os/0-core/core0/screen"
	"github.com/zero-os/0-core/core0/subsys/containers"
)

const (
	BaseMACAddress = "00:28:06:82:%02x:%02x"

	BaseIPAddr  = "172.19.%d.%d"
	metadataKey = "zero-os"
	metadataUri = "https://github.com/zero-os/0-core"
)

var (
	log = logging.MustGetLogger("kvm")
)

type LibvirtConnection struct {
	handler libvirt.DomainEventLifecycleCallback

	m    sync.Mutex
	conn *libvirt.Connect
}

type kvmManager struct {
	conmgr   containers.ContainerManager
	sequence uint16
	m        sync.Mutex
	libvirt  LibvirtConnection
	cell     *screen.RowCell
	evch     chan map[string]interface{}
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
	kvmCreateCommand          = "kvm.create"
	kvmPrepareMigrationTarget = "kvm.prepare_migration_target"
	kvmDestroyCommand         = "kvm.destroy"
	kvmShutdownCommand        = "kvm.shutdown"
	kvmRebootCommand          = "kvm.reboot"
	kvmResetCommand           = "kvm.reset"
	kvmPauseCommand           = "kvm.pause"
	kvmResumeCommand          = "kvm.resume"
	kvmInfoCommand            = "kvm.info"
	kvmInfoPSCommand          = "kvm.infops"
	kvmAttachDiskCommand      = "kvm.attach_disk"
	kvmDetachDiskCommand      = "kvm.detach_disk"
	kvmAddNicCommand          = "kvm.add_nic"
	kvmRemoveNicCommand       = "kvm.remove_nic"
	kvmLimitDiskIOCommand     = "kvm.limit_disk_io"
	kvmMigrateCommand         = "kvm.migrate"
	kvmListCommand            = "kvm.list"
	kvmMonitorCommand         = "kvm.monitor"
	kvmEventsCommand          = "kvm.events"

	DefaultBridgeName = "kvm0"
)

func KVMSubsystem(conmgr containers.ContainerManager, cell *screen.RowCell) error {
	if err := libvirt.EventRegisterDefaultImpl(); err != nil {
		return err
	}

	go func() {
		for {
			libvirt.EventRunDefaultImpl()
		}
	}()

	mgr := &kvmManager{
		conmgr: conmgr,
		cell:   cell,
		evch:   make(chan map[string]interface{}, 100), //buffer 100 event
	}

	mgr.libvirt.handler = mgr.handle

	cell.Text = "Virtual Machines: 0"
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
	pm.CmdMap[kvmInfoCommand] = process.NewInternalProcessFactory(mgr.info)
	pm.CmdMap[kvmInfoPSCommand] = process.NewInternalProcessFactory(mgr.infops)
	pm.CmdMap[kvmAttachDiskCommand] = process.NewInternalProcessFactory(mgr.attachDisk)
	pm.CmdMap[kvmDetachDiskCommand] = process.NewInternalProcessFactory(mgr.detachDisk)
	pm.CmdMap[kvmAddNicCommand] = process.NewInternalProcessFactory(mgr.addNic)
	pm.CmdMap[kvmRemoveNicCommand] = process.NewInternalProcessFactory(mgr.removeNic)
	pm.CmdMap[kvmLimitDiskIOCommand] = process.NewInternalProcessFactory(mgr.limitDiskIO)
	pm.CmdMap[kvmMigrateCommand] = process.NewInternalProcessFactory(mgr.migrate)
	pm.CmdMap[kvmListCommand] = process.NewInternalProcessFactory(mgr.list)
	pm.CmdMap[kvmPrepareMigrationTarget] = process.NewInternalProcessFactory(mgr.prepareMigrationTarget)

	//those next 2 commands should never be called by the client, unfortunately we don't have
	//support for internal commands yet.
	pm.CmdMap[kvmMonitorCommand] = process.NewInternalProcessFactory(mgr.monitor)
	pm.CmdMap[kvmEventsCommand] = process.NewInternalProcessFactoryWithCtx(mgr.events)

	//start domains monitoring command
	pm.GetManager().RunCmd(&core.Command{
		ID:              kvmMonitorCommand,
		Command:         kvmMonitorCommand,
		RecurringPeriod: 30,
	})

	//start events command
	pm.GetManager().RunCmd(&core.Command{
		ID:      kvmEventsCommand,
		Command: kvmEventsCommand,
	})

	return nil
}

type Media struct {
	URL    string         `json:"url"`
	Type   DiskDeviceType `json:"type"`
	Bus    string         `json:"bus"`
	IOTune *IOTuneParams  `json:"iotune,omitempty"`
}

type Nic struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	HWAddress string `json:"hwaddr"`
}
type NicParams struct {
	Nics []Nic       `json:"nics"`
	Port map[int]int `json:"port"`
}
type CreateParams struct {
	NicParams
	Name   string    `json:"name"`
	CPU    int       `json:"cpu"`
	Memory int       `json:"memory"`
	Media  []Media   `json:"media"`
	Tags   core.Tags `json:"tags"`
}

func (c *CreateParams) Valid() error {
	if err := c.NicParams.Valid(); err != nil {
		return err
	}
	if len(c.Media) < 1 {
		return fmt.Errorf("At least a boot disk has to be provided")
	}
	if c.CPU == 0 {
		return fmt.Errorf("CPU is a required parameter")
	}
	if c.Memory == 0 {
		return fmt.Errorf("Memory is a required parameter")
	}
	return nil
}

func (c *NicParams) Valid() error {
	brcounter := make(map[string]int)
	for _, nic := range c.Nics {
		switch nic.Type {
		case "default":
			brcounter[DefaultBridgeName]++
			if brcounter[DefaultBridgeName] > 1 {
				return fmt.Errorf("only one default network is allowed")
			}
		case "bridge":
			if nic.ID == DefaultBridgeName {
				return fmt.Errorf("cannot use bridge %s with nic type 'bridge', please use type default instead", DefaultBridgeName)
			}
			brcounter[nic.ID]++
			if brcounter[nic.ID] > 1 {
				return fmt.Errorf("connecting to bridge '%s' more than one time is not allowed", nic.ID)
			}
		case "vlan":
		case "vxlan":
		default:
			return fmt.Errorf("invalid nic type '%s'", nic.Type)
		}
	}
	return nil
}

type DomainUUID struct {
	UUID string `json:"uuid"`
}

type ManDiskParams struct {
	UUID  string `json:"uuid"`
	Media Media  `json:"media"`
}

type ManNicParams struct {
	Nic
	UUID string `json:"uuid"`
}

type MigrateParams struct {
	UUID    string `json:"uuid"`
	DestURI string `json:"desturi"`
}

type IOTuneParams struct {
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

type LimitDiskIOParams struct {
	IOTuneParams
	UUID  string `json:"uuid"`
	Media Media  `json:"media"`
}

type DomainStats struct {
	Vcpu  []DomainStatsVcpu  `json"vcpu"`
	Net   []DomainStatsNet   `json"net"`
	Block []DomainStatsBlock `json"block"`
}

type DomainStatsVcpu struct {
	State int    `json"state"`
	Time  uint64 `json"time"`
}

type DomainStatsNet struct {
	Name    string `json"name"`
	RxBytes uint64 `json"rxbytes"`
	RxPkts  uint64 `json"rxpkts"`
	RxErrs  uint64 `json"rxerrs"`
	RxDrop  uint64 `json"rxdrop"`
	TxBytes uint64 `json"txbytes"`
	TxPkts  uint64 `json"txpkts"`
	TxErrs  uint64 `json"txerrs"`
	TxDrop  uint64 `json"txdrop"`
}

type DomainStatsBlock struct {
	Name    string `json"name"`
	RdBytes uint64 `json"rdbytes"`
	RdTimes uint64 `json"rdtimes"`
	WrBytes uint64 `json"wrbytes"`
	WrTimes uint64 `json"wrtimes"`
}

type LastStatistics struct {
	Last  float64 `json:"last_value"`
	Epoch int64   `json:"last_time"`
}

type QemuImgInfoResult struct {
	Format string `json:"format"`
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

func IOTuneParamsToIOTune(inp IOTuneParams) IOTune {
	out := IOTune{}
	if inp.TotalBytesSecSet {
		out.TotalBytesSec = &inp.TotalBytesSec
	}
	if inp.ReadBytesSecSet {
		out.ReadBytesSec = &inp.ReadBytesSec
	}
	if inp.WriteBytesSecSet {
		out.WriteBytesSec = &inp.WriteBytesSec
	}
	if inp.TotalIopsSecSet {
		out.TotalIopsSec = &inp.TotalIopsSec
	}
	if inp.ReadIopsSecSet {
		out.ReadIopsSec = &inp.ReadIopsSec
	}
	if inp.WriteIopsSecSet {
		out.WriteIopsSec = &inp.WriteIopsSec
	}
	if inp.TotalBytesSecMaxSet {
		out.TotalBytesSecMax = &inp.TotalBytesSecMax
	}
	if inp.ReadBytesSecMaxSet {
		out.ReadBytesSecMax = &inp.ReadBytesSecMax
	}
	if inp.WriteBytesSecMaxSet {
		out.WriteBytesSecMax = &inp.WriteBytesSecMax
	}
	if inp.TotalIopsSecMaxSet {
		out.TotalIopsSecMax = &inp.TotalIopsSecMax
	}
	if inp.ReadIopsSecMaxSet {
		out.ReadIopsSecMax = &inp.ReadIopsSecMax
	}
	if inp.WriteIopsSecMaxSet {
		out.WriteIopsSecMax = &inp.WriteIopsSecMax
	}
	if inp.TotalBytesSecMaxLengthSet {
		out.TotalBytesSecMaxLength = &inp.TotalBytesSecMaxLength
	}
	if inp.ReadBytesSecMaxLengthSet {
		out.ReadBytesSecMaxLength = &inp.ReadBytesSecMaxLength
	}
	if inp.WriteBytesSecMaxLengthSet {
		out.WriteBytesSecMaxLength = &inp.WriteBytesSecMaxLength
	}
	if inp.TotalIopsSecMaxLengthSet {
		out.TotalIopsSecMaxLength = &inp.TotalIopsSecMaxLength
	}
	if inp.ReadIopsSecMaxLengthSet {
		out.ReadIopsSecMaxLength = &inp.ReadIopsSecMaxLength
	}
	if inp.WriteIopsSecMaxLengthSet {
		out.WriteIopsSecMaxLength = &inp.WriteIopsSecMaxLength
	}
	if inp.SizeIopsSecSet {
		out.SizeIopsSec = &inp.SizeIopsSec
	}
	if inp.GroupNameSet {
		out.GroupName = &inp.GroupName
	}
	return out
}

func (c *LibvirtConnection) register(conn *libvirt.Connect) {
	_, err := conn.DomainEventLifecycleRegister(nil, c.handler)
	if err != nil {
		log.Errorf("failed to regist event handler: %s", err)
	}
}

func (c *LibvirtConnection) getConnection() (*libvirt.Connect, error) {
	c.m.Lock()
	defer c.m.Unlock()
	if c.conn != nil {
		if alive, err := c.conn.IsAlive(); err == nil && alive == true {
			return c.conn, nil
		}

		c.conn.Close()
	}

	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return nil, err
	}

	c.register(conn)
	c.conn = conn
	return c.conn, nil
}

func (m *kvmManager) getDomainStruct(uuid string) (*Domain, error) {
	conn, err := m.libvirt.getConnection()
	if err != nil {
		return nil, err
	}
	domain, err := conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return nil, fmt.Errorf("couldn't find domain with the uuid %s", uuid)
	}
	domainxml, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_SECURE)
	if err != nil {
		return nil, fmt.Errorf("cannot get domain xml: %v", err)
	}
	domainstruct := Domain{}
	err = xml.Unmarshal([]byte(domainxml), &domainstruct)
	if err != nil {
		return nil, fmt.Errorf("cannot parse the domain xml: %v", err)
	}
	return &domainstruct, nil
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
			Driver: DiskDriver{
				Cache: "none",
			},
			Source: DiskSource{
				Protocol: "nbd",
				Name:     name,
				Host: DiskSourceNetworkHost{
					Port: port,
					Name: u.Hostname(),
				},
			},
		}
	case "nbd+unix":
		return DiskDevice{
			Type: DiskTypeNetwork,
			Target: DiskTarget{
				Dev: target,
			},
			Driver: DiskDriver{
				Cache: "none",
			},
			Source: DiskSource{
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

func getDiskType(path string) string {
	result, err := pm.GetManager().System("qemu-img", "info", "--output=json", path)
	if err != nil {
		return "raw"
	}
	var params QemuImgInfoResult
	if err := json.Unmarshal([]byte(result.Streams.Stdout()), &params); err != nil {
		return "raw"
	}
	return params.Format
}

func (m *kvmManager) mkFileDisk(idx int, u *url.URL) DiskDevice {
	target := "vd" + string(97+idx)
	return DiskDevice{
		Type: DiskTypeFile,
		Target: DiskTarget{
			Dev: target,
		},
		Source: DiskSource{
			File: u.String(),
		},
		Driver: DiskDriver{
			Type: DiskDriverType(getDiskType(u.Path)),
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

	if media.IOTune != nil {
		disk.IOTune = IOTuneParamsToIOTune(*media.IOTune)
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
			Unit:     "MiB",
		},
		VCPU: params.CPU,
		OS: OS{
			Type: OSType{
				Type: OSTypeTypeHVM,
				Arch: ArchX86_64,
			},
		},
		Features: FeaturesType{},
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
			},
			Graphics: []GraphicsDevice{
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

	for idx, media := range params.Media {
		domain.Devices.Disks = append(domain.Devices.Disks, m.mkDisk(idx, media))
	}

	return &domain, nil
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

func (m *kvmManager) updateView() {
	conn, err := m.libvirt.getConnection()
	if err != nil {
		return
	}
	domains, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return
	}

	m.cell.Text = fmt.Sprintf("Virtual Machines: %d", len(domains))
	screen.Refresh()
}

func (m *kvmManager) create(cmd *core.Command) (interface{}, error) {
	defer m.updateView()
	var params CreateParams
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}

	params.Tags = cmd.Tags
	if err := params.Valid(); err != nil {
		return nil, err
	}

	seq := m.getNextSequence()

	domain, err := m.mkDomain(seq, &params)
	if err != nil {
		return nil, err
	}

	if err := m.setNetworking(&params.NicParams, seq, domain); err != nil {
		return nil, err
	}

	data, err := xml.MarshalIndent(domain, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to generate domain xml: %s", err)
	}

	conn, err := m.libvirt.getConnection()
	if err != nil {
		return nil, err
	}
	//create domain
	_, err = conn.DomainCreateXML(string(data), libvirt.DOMAIN_NONE)
	if err != nil {
		return nil, fmt.Errorf("failed to create machine: %s", err)
	}

	dom, err := conn.LookupDomainByUUIDString(domain.UUID)
	if err != nil {
		return nil, fmt.Errorf("couldn't find domain with the uuid %s", domain.UUID)
	}

	tags, err := json.Marshal(&params.Tags)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal tags for domain with the uuid %s", domain.UUID)
	}

	metaData := MetaData{Value: string(tags)}
	metaXML, err := xml.Marshal(&metaData)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal metadata for domain with the uuid %s", domain.UUID)
	}

	err = dom.SetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, string(metaXML), metadataKey, metadataUri, libvirt.DOMAIN_AFFECT_LIVE)
	if err != nil {
		return nil, fmt.Errorf("couldn't set metadata for domain with the uuid %s", domain.UUID)
	}

	return domain.UUID, nil
}

func (m *kvmManager) prepareMigrationTarget(cmd *core.Command) (interface{}, error) {
	defer m.updateView()
	var params struct {
		NicParams
		UUID string `json:"uuid"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}

	var domain Domain
	domain.UUID = params.UUID
	if err := params.NicParams.Valid(); err != nil {
		return nil, err
	}

	seq := m.getNextSequence()
	if err := m.setNetworking(&params.NicParams, seq, &domain); err != nil {
		return nil, err
	}
	return nil, nil
}

func (m *kvmManager) getDomain(cmd *core.Command) (*libvirt.Domain, string, error) {
	var params DomainUUID
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, "", err
	}

	conn, err := m.libvirt.getConnection()
	if err != nil {
		return nil, "", err
	}
	domain, err := conn.LookupDomainByUUIDString(params.UUID)
	if err != nil {
		return nil, params.UUID, fmt.Errorf("couldn't find domain with the uuid %s", params.UUID)
	}
	return domain, params.UUID, err
}

func (m *kvmManager) destroy(cmd *core.Command) (interface{}, error) {
	defer m.updateView()
	domain, uuid, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	if err := domain.Destroy(); err != nil {
		return nil, fmt.Errorf("failed to destroy machine: %s", err)
	}
	m.unPortForward(uuid)

	return nil, nil
}

func (m *kvmManager) shutdown(cmd *core.Command) (interface{}, error) {
	domain, uuid, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	if err := domain.Shutdown(); err != nil {
		return nil, fmt.Errorf("failed to shutdown machine: %s", err)
	}

	m.unPortForward(uuid)

	return nil, nil
}

func (m *kvmManager) reboot(cmd *core.Command) (interface{}, error) {
	domain, _, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	if err := domain.Reboot(libvirt.DOMAIN_REBOOT_DEFAULT); err != nil {
		return nil, fmt.Errorf("failed to reboot machine: %s", err)
	}

	return nil, nil
}

func (m *kvmManager) reset(cmd *core.Command) (interface{}, error) {
	domain, _, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	if err := domain.Reset(0); err != nil {
		return nil, fmt.Errorf("failed to reset machine: %s", err)
	}

	return nil, nil
}

func (m *kvmManager) pause(cmd *core.Command) (interface{}, error) {
	domain, _, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	if err := domain.Suspend(); err != nil {
		return nil, fmt.Errorf("failed to pause machine: %s", err)
	}

	return nil, nil
}

func (m *kvmManager) resume(cmd *core.Command) (interface{}, error) {
	domain, _, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	if err := domain.Resume(); err != nil {
		return nil, fmt.Errorf("failed to resume machine: %s", err)
	}

	return nil, nil
}

func (m *kvmManager) info(cmd *core.Command) (interface{}, error) {
	domain, _, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	conn, err := m.libvirt.getConnection()
	if err != nil {
		return nil, err
	}
	infos, err := conn.GetAllDomainStats([]*libvirt.Domain{domain}, libvirt.DOMAIN_STATS_STATE|libvirt.DOMAIN_STATS_VCPU|libvirt.DOMAIN_STATS_INTERFACE|libvirt.DOMAIN_STATS_BLOCK,
		libvirt.CONNECT_GET_ALL_DOMAINS_STATS_ACTIVE|libvirt.CONNECT_GET_ALL_DOMAINS_STATS_INACTIVE)
	if err != nil {
		return nil, fmt.Errorf("failed to get machine info: %s", err)
	}
	info := infos[0]
	cpus := make([]DomainStatsVcpu, len(info.Vcpu))
	for i, vcpu := range info.Vcpu {
		cpus[i] = DomainStatsVcpu{
			State: int(vcpu.State),
			Time:  vcpu.Time,
		}
	}
	nets := make([]DomainStatsNet, len(info.Net))
	for i, net := range info.Net {
		nets[i] = DomainStatsNet{
			Name:    net.Name,
			RxBytes: net.RxBytes,
			RxPkts:  net.RxPkts,
			RxErrs:  net.RxErrs,
			RxDrop:  net.RxDrop,
			TxBytes: net.TxBytes,
			TxPkts:  net.TxPkts,
			TxErrs:  net.TxErrs,
			TxDrop:  net.TxDrop,
		}
	}
	blocks := make([]DomainStatsBlock, len(info.Block))
	for i, block := range info.Block {
		blocks[i] = DomainStatsBlock{
			Name:    block.Name,
			RdBytes: block.RdBytes,
			RdTimes: block.RdTimes,
			WrBytes: block.WrBytes,
			WrTimes: block.WrTimes,
		}
	}
	stat := DomainStats{
		Vcpu:  cpus,
		Net:   nets,
		Block: blocks,
	}
	return stat, nil
}

func (m *kvmManager) attachDevice(uuid, xml string) error {
	conn, err := m.libvirt.getConnection()
	if err != nil {
		return err
	}
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
	conn, err := m.libvirt.getConnection()
	if err != nil {
		return err
	}
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
	domainstruct, err := m.getDomainStruct(params.UUID)
	if err != nil {
		return nil, err
	}
	count := len(domainstruct.Devices.Disks)
	disk := m.mkDisk(count, params.Media)
	disks := domainstruct.Devices.Disks
	for _, d := range disks {
		if d.Source == disk.Source {
			return nil, fmt.Errorf("The disk you tried is already attached to the vm")
		}
	}
	diskxml, err := xml.MarshalIndent(disk, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot marshal disk to xml")
	}
	return nil, m.attachDevice(params.UUID, string(diskxml[:]))
}

func (m *kvmManager) detachDisk(cmd *core.Command) (interface{}, error) {
	var (
		params ManDiskParams
		disk   *DiskDevice
	)
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}
	domainstruct, err := m.getDomainStruct(params.UUID)
	if err != nil {
		return nil, err
	}
	disks := domainstruct.Devices.Disks
	inp := m.mkDisk(0, params.Media)
	for _, d := range disks {
		if d.Source == inp.Source {
			disk = &d
			break
		}
	}
	if disk == nil {
		return nil, fmt.Errorf("The disk you tried is not attached to the vm")
	}
	diskxml, err := xml.MarshalIndent(disk, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot marshal disk to xml")
	}
	return nil, m.detachDevice(params.UUID, string(diskxml[:]))
}

func (m *kvmManager) addNic(cmd *core.Command) (interface{}, error) {
	var (
		params ManNicParams
		inf    *InterfaceDevice
		err    error
	)
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}
	nic := Nic{
		Type:      params.Type,
		ID:        params.ID,
		HWAddress: params.HWAddress,
	}

	domainstruct, err := m.getDomainStruct(params.UUID)
	if err != nil {
		return nil, err
	}

	switch nic.Type {
	case "default":
		for _, nic := range domainstruct.Devices.Interfaces {
			if nic.Source.Bridge == DefaultBridgeName {
				return nil, fmt.Errorf("The default nic is already attached to the vm")
			}
		}
		seq := m.getNextSequence()
		// TODO: use the ports that the domain was created with initially
		inf, err = m.prepareDefaultNetwork(params.UUID, seq, map[int]int{})
	case "bridge":
		if nic.ID == DefaultBridgeName {
			err = fmt.Errorf("the default bridge for the vm should not be added manually")
		} else {
			inf, err = m.prepareBridgeNetwork(&nic)
		}
	case "vlan":
		inf, err = m.prepareVLanNetwork(&nic)
	case "vxlan":
		inf, err = m.prepareVXLanNetwork(&nic)
	default:
		err = fmt.Errorf("unsupported network mode: %s", nic.Type)
	}
	if err != nil {
		return nil, err
	}

	// We check for the default network upfront
	if nic.Type != "default" {
		for _, nic := range domainstruct.Devices.Interfaces {
			if nic.Source == inf.Source {
				return nil, fmt.Errorf("This nic is already attached to the vm")
			}
		}
	}

	ifxml, err := xml.MarshalIndent(inf, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot marshal nic to xml")
	}
	return nil, m.attachDevice(params.UUID, string(ifxml[:]))
}

func (m *kvmManager) removeNic(cmd *core.Command) (interface{}, error) {
	var (
		params ManNicParams
		inf    *InterfaceDevice
		tmp    *InterfaceDevice
		source InterfaceDeviceSource
		err    error
	)
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}
	nic := Nic{
		Type:      params.Type,
		ID:        params.ID,
		HWAddress: params.HWAddress,
	}

	switch nic.Type {
	case "default":
		source = InterfaceDeviceSource{
			Bridge: DefaultBridgeName,
		}
	case "bridge":
		source = InterfaceDeviceSource{
			Bridge: nic.ID,
		}
	case "vlan":
		tmp, err = m.prepareVLanNetwork(&nic)
		source = tmp.Source
	case "vxlan":
		tmp, err = m.prepareVXLanNetwork(&nic)
		source = tmp.Source
	default:
		err = fmt.Errorf("unsupported network mode: %s", nic.Type)
	}
	if err != nil {
		return nil, err
	}

	domainstruct, err := m.getDomainStruct(params.UUID)
	if err != nil {
		return nil, err
	}

	for _, nic := range domainstruct.Devices.Interfaces {
		if nic.Source == source {
			inf = &nic
		}
	}
	if inf == nil {
		return nil, fmt.Errorf("The nic you tried is not attached to the vm")
	}

	ifxml, err := xml.MarshalIndent(inf, "", "  ")
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
	conn, err := m.libvirt.getConnection()
	if err != nil {
		return nil, err
	}
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
	domainstruct, err := m.getDomainStruct(params.UUID)
	if err != nil {
		return nil, err
	}
	disks := domainstruct.Devices.Disks
	inp := m.mkDisk(0, params.Media)
	target := ""
	for _, d := range disks {
		if d.Source == inp.Source {
			target = d.Target.Dev
			break
		}
	}
	if target == "" {
		return nil, fmt.Errorf("The disk you tried is not attached to the vm")
	}
	if err := domain.SetBlockIoTune(target, &blockParams, libvirt.DOMAIN_AFFECT_LIVE); err != nil {
		return nil, fmt.Errorf("failed to tune disk: %s", err)
	}
	return nil, nil
}

func (m *kvmManager) migrate(cmd *core.Command) (interface{}, error) {
	domain, _, err := m.getDomain(cmd)
	if err != nil {
		return nil, err
	}
	var params MigrateParams
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}
	name, err := domain.GetName()
	if err != nil {
		return nil, err
	}
	if err = domain.MigrateToURI(params.DestURI, libvirt.MIGRATE_LIVE|libvirt.MIGRATE_UNDEFINE_SOURCE|libvirt.MIGRATE_PEER2PEER|libvirt.MIGRATE_TUNNELLED, name, 10000000000); err != nil {
		return nil, err
	}
	return nil, nil
}

type Machine struct {
	ID         int       `json:"id"`
	UUID       string    `json:"uuid"`
	Name       string    `json:"name"`
	State      string    `json:"state"`
	Vnc        int       `json:"vnc"`
	Tags       core.Tags `json:"tags"`
	IfcTargets []string  `json:"ifctargets"`
}

func (m *kvmManager) list(cmd *core.Command) (interface{}, error) {
	conn, err := m.libvirt.getConnection()
	if err != nil {
		return nil, err
	}
	domains, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return nil, fmt.Errorf("failed to list machines: %s", err)
	}

	found := make([]Machine, 0)

	for _, domain := range domains {
		uuid, err := domain.GetUUIDString()
		if err != nil {
			return nil, err
		}
		domainstruct, err := m.getDomainStruct(uuid)
		if err != nil {
			return nil, err
		}
		id, err := domain.GetID()
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
		port := -1
		for _, graphics := range domainstruct.Devices.Graphics {
			if graphics.Type == GraphicsDeviceTypeVNC {
				port = graphics.Port
				break
			}
		}

		targets := []string{}
		for _, ifc := range domainstruct.Devices.Interfaces {
			targets = append(targets, ifc.Target.Dev)

		}

		domainMetaData, err := domain.GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, metadataUri, libvirt.DOMAIN_AFFECT_LIVE)
		if err != nil {
			return nil, fmt.Errorf("couldn't get metadata for domain with the uuid %s", uuid)
		}

		var metaData MetaData
		err = xml.Unmarshal([]byte(domainMetaData), &metaData)
		if err != nil {
			return nil, fmt.Errorf("couldn't xml unmarshal metadata for domain with the uuid %s", uuid)
		}
		var tags core.Tags
		err = json.Unmarshal([]byte(metaData.Value), &tags)
		if err != nil {
			return nil, fmt.Errorf("couldn't json unmarshal tags for domain with the uuid %s", uuid)
		}

		found = append(found, Machine{
			ID:         int(id),
			UUID:       uuid,
			Name:       name,
			State:      StateToString(state),
			Vnc:        port,
			Tags:       tags,
			IfcTargets: targets,
		})
	}

	return found, nil
}

func (m *kvmManager) monitor(cmd *core.Command) (interface{}, error) {
	conn, err := m.libvirt.getConnection()
	if err != nil {
		return nil, err
	}
	infos, err := conn.GetAllDomainStats(nil, libvirt.DOMAIN_STATS_STATE|libvirt.DOMAIN_STATS_VCPU|libvirt.DOMAIN_STATS_INTERFACE|libvirt.DOMAIN_STATS_BLOCK,
		libvirt.CONNECT_GET_ALL_DOMAINS_STATS_ACTIVE)
	if err != nil {
		return nil, err
	}

	p := pm.GetManager()
	for _, info := range infos {
		name, err := info.Domain.GetName()
		if err != nil {
			return nil, err
		}

		for i, vcpu := range info.Vcpu {
			nr := fmt.Sprintf("%d", i)
			p.Aggregate(
				pm.AggreagteAverage,
				"kvm.vcpu.state", float64(vcpu.State), name,
				pm.Tag{"type", "virt"}, pm.Tag{"nr", nr},
			)

			p.Aggregate(
				pm.AggreagteAverage,
				"kvm.vcpu.time", float64(vcpu.Time)/1000000000., name,
				pm.Tag{"type", "virt"}, pm.Tag{"nr", nr},
			)
		}

		for _, net := range info.Net {
			p.Aggregate(
				pm.AggreagteDifference,
				"kvm.net.rxbytes", float64(net.RxBytes), name,
				pm.Tag{"type", "virt"}, pm.Tag{"name", net.Name},
			)

			p.Aggregate(
				pm.AggreagteDifference,
				"kvm.net.rxpackets", float64(net.RxPkts), name,
				pm.Tag{"type", "virt"}, pm.Tag{"name", net.Name},
			)

			p.Aggregate(
				pm.AggreagteDifference,
				"kvm.net.txbytes", float64(net.TxBytes), name,
				pm.Tag{"type", "virt"}, pm.Tag{"name", net.Name},
			)

			p.Aggregate(
				pm.AggreagteDifference,
				"kvm.net.txpackets", float64(net.TxPkts), name,
				pm.Tag{"type", "virt"}, pm.Tag{"name", net.Name},
			)
		}

		for _, block := range info.Block {
			p.Aggregate(
				pm.AggreagteDifference,
				"kvm.disk.rdbytes", float64(block.RdBytes), uuid,
				pm.Tag{"type", "virt"}, pm.Tag{"name", block.Name},
			)

			p.Aggregate(
				pm.AggreagteDifference,
				"kvm.disk.rdtimes", float64(block.RdTimes), uuid,
				pm.Tag{"type", "virt"}, pm.Tag{"name", block.Name},
			)

			p.Aggregate(
				pm.AggreagteDifference,
				"kvm.disk.wrbytes", float64(block.WrBytes), uuid,
				pm.Tag{"type", "virt"}, pm.Tag{"name", block.Name},
			)

			p.Aggregate(
				pm.AggreagteDifference,
				"kvm.disk.wrtimes", float64(block.WrTimes), uuid,
				pm.Tag{"type", "virt"}, pm.Tag{"name", block.Name},
			)
		}
	}

	return nil, nil
}

func (m *kvmManager) infops(cmd *core.Command) (interface{}, error) {
	var params DomainUUID
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}

	runner, err := pm.GetManager().RunCmd(&core.Command{
		ID:      uuid.New(),
		Command: "aggregator.query",
		Arguments: core.MustArguments(core.M{
			//todo: add support to partial key match maybe so we can do 'kvm.*'?
			"tags": core.M{
				"id": params.UUID,
			},
		}),
	})
	if err != nil {
		return nil, err
	}

	result := runner.Wait()
	if result.State != core.StateSuccess {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal([]byte(result.Data), &data); err != nil {
		return nil, err
	}

	return data, nil
}
