// +build amd64

package kvm

import (
	"encoding/xml"
)

/*
<domain type='kvm'>
  <name>demo2</name>
  <uuid>4dea24b3-1d52-d8f3-2516-782e98a23fa0</uuid>
  <memory>131072</memory>
  <vcpu>1</vcpu>
  <os>
    <type arch="i686">hvm</type>
  </os>
  <clock sync="localtime"/>
  <devices>
    <emulator>/usr/bin/qemu-kvm</emulator>
    <disk type='file' device='disk'>
      <source file='/var/lib/libvirt/images/demo2.img'/>
      <target dev='hda'/>
    </disk>
    <interface type='network'>
      <source network='default'/>
      <mac address='24:42:53:21:52:45'/>
    </interface>
    <graphics type='vnc' port='-1' keymap='de'/>
  </devices>
</domain>
*/

type DomainType string
type OSTypeType string

const (
	DomainTypeKVM = "kvm"

	OSTypeTypeHVM OSTypeType = "hvm"

	ArchI686   = "i686"
	ArchX86_64 = "x86_64"
)

type OSType struct {
	Type OSTypeType `xml:",chardata"`
	Arch string     `xml:"arch,attr"`
}

type FeaturesType struct {
	Acpi string `xml:"acpi"`
	Apic string `xml:"apic"`
	Pae  string `xml:"pae"`
}

type OS struct {
	Type    OSType `xml:"type"`
	Kernel  string `xml:"kernel,omitempty"`
	InitRD  string `xml:"initrd,omitempty"`
	Cmdline string `xml:"cmdline,omitempty"`
}

type Memory struct {
	Capacity int    `xml:",chardata"`
	Unit     string `xml:"unit,attr,omitempty"`
}

type Device interface{}

type QemuArg struct {
	XMLName xml.Name `xml:"qemu:arg"`
	Value   string   `xml:"value,attr"`
}
type Qemu struct {
	Args []QemuArg
}

type Domain struct {
	XMLName  xml.Name     `xml:"domain"`
	QemuNS   string       `xml:"xmlns:qemu,attr"`
	Type     DomainType   `xml:"type,attr"`
	Name     string       `xml:"name"`
	UUID     string       `xml:"uuid"`
	Memory   Memory       `xml:"memory"`
	VCPU     int          `xml:"vcpu"`
	OS       OS           `xml:"os"`
	Features FeaturesType `xml:"features"`
	Devices  Devices      `xml:"devices"`
	Qemu     Qemu         `xml:"qemu:commandline"`
}

type DiskType string
type DiskDeviceType string
type DiskDriverType string

const (
	DiskTypeFile    DiskType = "file"
	DiskTypeDir     DiskType = "dir"
	DiskTypeVolume  DiskType = "volume"
	DiskTypeNetwork DiskType = "network"

	DiskDeviceTypeDisk  DiskDeviceType = "disk"
	DiskDeviceTypeCDROM DiskDeviceType = "cdrom"
)

type Devices struct {
	Emulator    string            `xml:"emulator"`
	Graphics    []GraphicsDevice  `xml:"graphics"`
	Disks       []DiskDevice      `xml:"disk"`
	Interfaces  []InterfaceDevice `xml:"interface"`
	Devices     []Device          `xml:"device"`
	Filesystems []Filesystem      `xml:"filesystem"`
}

type FilesystemDir struct {
	Dir string `xml:"dir,attr"`
}

type Bool struct{}

type Filesystem struct {
	Source   FilesystemDir `xml:"source"`
	Target   FilesystemDir `xml:"target"`
	Readonly *Bool         `xml:"readonly"`
}

type DiskSource struct {
	// File
	File string `xml:"file,attr,omitempty"`
	// Block
	Dev string `xml:"dev,attr.,omitempty"`
	// Network
	Protocol string                `xml:"protocol,attr,omitempty"`
	Host     DiskSourceNetworkHost `xml:"host,omitempty"`
	Name     string                `xml:"name,attr,omitempty,omitempty"`
}

type DiskSourceNetworkHost struct {
	Transport string `xml:"transport,attr,omitempty"`
	Socket    string `xml:"socket,attr,omitempty"`
	Port      string `xml:"port,attr,omitempty"`
	Name      string `xml:"name,attr,omitempty"`
}

type DiskTarget struct {
	Dev string `xml:"dev,attr"`
	Bus string `xml:"bus,attr"`
}

type DiskDriver struct {
	Type  DiskDriverType `xml:"type,attr,omitempty"`
	Cache string         `xml:"cache,attr,omitempty"`
	IO    string         `xml:"io,attr,omitempty"`
}

type DiskDevice struct {
	XMLName xml.Name       `xml:"disk"`
	Type    DiskType       `xml:"type,attr"`
	Device  DiskDeviceType `xml:"device,attr"`
	Source  DiskSource     `xml:"source"`
	Target  DiskTarget     `xml:"target"`
	Driver  DiskDriver     `xml:"driver"`
	IOTune  IOTune         `xml:"iotune,omitempty"`
	Alias   Alias          `xml:"alias"`
}

type IOTune struct {
	TotalBytesSec          *uint64 `xml:"total_bytes_sec,omitempty"`
	ReadBytesSec           *uint64 `xml:"read_bytes_sec,omitempty"`
	WriteBytesSec          *uint64 `xml:"write_bytes_sec,omitempty"`
	TotalIopsSec           *uint64 `xml:"total_iops_sec,omitempty"`
	ReadIopsSec            *uint64 `xml:"read_iops_sec,omitempty"`
	WriteIopsSec           *uint64 `xml:"write_iops_sec,omitempty"`
	TotalBytesSecMax       *uint64 `xml:"total_bytes_sec_max,omitempty"`
	ReadBytesSecMax        *uint64 `xml:"read_bytes_sec_max,omitempty"`
	WriteBytesSecMax       *uint64 `xml:"write_bytes_sec_max,omitempty"`
	TotalIopsSecMax        *uint64 `xml:"total_iops_sec_max,omitempty"`
	ReadIopsSecMax         *uint64 `xml:"read_iops_sec_max,omitempty"`
	WriteIopsSecMax        *uint64 `xml:"write_iops_sec_max,omitempty"`
	TotalBytesSecMaxLength *uint64 `xml:"total_bytes_sec_max_length,omitempty"`
	ReadBytesSecMaxLength  *uint64 `xml:"read_bytes_sec_max_length,omitempty"`
	WriteBytesSecMaxLength *uint64 `xml:"write_bytes_sec_max_length,omitempty"`
	TotalIopsSecMaxLength  *uint64 `xml:"total_iops_sec_max_length,omitempty"`
	ReadIopsSecMaxLength   *uint64 `xml:"read_iops_sec_max_length,omitempty"`
	WriteIopsSecMaxLength  *uint64 `xml:"write_iops_sec_max_length,omitempty"`
	SizeIopsSec            *uint64 `xml:"size_iops_sec,omitempty"`
	GroupName              *string `xml:"group_name,omitempty"`
}

type GraphicsDeviceType string

const (
	GraphicsDeviceTypeVNC GraphicsDeviceType = "vnc"
)

type Listen struct {
	Type    string `xml:"type,attr"`
	Address string `xml:"address,attr"`
}

type GraphicsDevice struct {
	XMLName xml.Name           `xml:"graphics"`
	Type    GraphicsDeviceType `xml:"type,attr"`
	Port    int                `xml:"port,attr"`
	KeyMap  string             `xml:"keymap,attr"`
	Listen  Listen             `xml:"listen"`
}

type InterfaceDeviceType string

const (
	InterfaceDeviceTypeBridge  InterfaceDeviceType = "bridge"
	InterfaceDeviceTypeNetwork InterfaceDeviceType = "network"
)

type InterfaceDeviceSource struct {
	Bridge  string `xml:"bridge,attr,omitempty"`
	Network string `xml:"network,attr,omitempty"`
}

type InterfaceDeviceTarget struct {
	Dev string `xml:"dev,attr"`
}

type InterfaceDeviceModel struct {
	Type string `xml:"type,attr"`
}

type InterfaceDeviceMac struct {
	Address string `xml:"address,attr"`
}

type Alias struct {
	Name string `xml:"name,attr"`
}
type InterfaceDevice struct {
	XMLName xml.Name              `xml:"interface"`
	Type    InterfaceDeviceType   `xml:"type,attr"`
	Source  InterfaceDeviceSource `xml:"source"`
	Target  InterfaceDeviceTarget `xml:"target,omitempty"`
	Model   InterfaceDeviceModel  `xml:"model"`
	Alias   Alias                 `xml:"alias"`
	Mac     *InterfaceDeviceMac   `xml:"mac,omitempty"`
}

type SerialDeviceType string

const (
	SerialDeviceTypePTY SerialDeviceType = "pty"
)

type SerialSource struct {
	XMLName xml.Name `xml:"source"`
	Path    string   `xml:"path,attr"`
}

type SerialTarget struct {
	XMLName xml.Name `xml:"target"`
	Port    int      `xml:"port,attr"`
}

type SerialAlias struct {
	XMLName xml.Name `xml:"alias"`
	Name    string   `xml:"name,attr"`
}

type ConsoleTarget struct {
	XMLName xml.Name `xml:"target"`
	Port    int      `xml:"port,attr"`
	Type    string   `xml:"type,attr"`
}

type SerialDevice struct {
	XMLName xml.Name         `xml:"serial"`
	Type    SerialDeviceType `xml:"type,attr"`
	Source  SerialSource     `xml:"source"`
	Target  SerialTarget     `xml:"target"`
	Alias   SerialAlias      `xml:"alias"`
}
type ConsoleDevice struct {
	XMLName xml.Name         `xml:"console"`
	Type    SerialDeviceType `xml:"type,attr"`
	TTY     string           `xml:"tty,attr"`
	Source  SerialSource     `xml:"source"`
	Target  ConsoleTarget    `xml:"target"`
	Alias   SerialAlias      `xml:"alias"`
}

type Network struct {
	XMLName xml.Name `xml:"network"`
	Name    string   `xml:"name"`
	Forward struct {
		Mode string `xml:"mode,attr"`
	} `xml:"forward"`
	Bridge struct {
		Name string `xml:"name,attr"`
	} `xml:"bridge"`
	VirtualPort struct {
		Type string `xml:"type,attr"`
	} `xml:"virtualport"`
}
