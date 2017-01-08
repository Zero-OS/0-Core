package kvm

import "encoding/xml"

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

type OS struct {
	Type OSType `xml:"type"`
}

type Memory struct {
	Capacity int    `xml:",chardata"`
	Unit     string `xml:"unit,attr,omitempty"`
}

type Device interface{}

type Domain struct {
	XMLName xml.Name   `xml:"domain"`
	Type    DomainType `xml:"type,attr"`
	Name    string     `xml:"name"`
	UUID    string     `xml:"uuid"`
	Memory  Memory     `xml:"memory"`
	VCPU    int        `xml:"vcpu`
	OS      OS         `xml:"os"`
	Devices Devices    `xml:"devices"`
}

type DiskType string
type DiskDeviceType string

const (
	DiskTypeFile    DiskType = "file"
	DiskTypeDir     DiskType = "dir"
	DiskTypeVolume  DiskType = "volume"
	DiskTypeNetwork DiskType = "network"

	DiskDeviceTypeDisk  DiskDeviceType = "disk"
	DiskDeviceTypeCDROM DiskDeviceType = "cdrom"
)

type Devices struct {
	Emulator string `xml:"emulator"`
	Devices  []Device
}

type DiskSource interface{}

type DiskSourceFile struct {
	File string `xml:"file,attr"`
}

type DiskSourceBlock struct {
	Dev string `xml:"dev,attr"`
}

type DiskTarget struct {
	Dev string `xml:"dev,attr"`
	Bus string `xml:"bus,attr"`
}

type DiskDevice struct {
	XMLName xml.Name       `xml:"disk"`
	Type    DiskType       `xml:"type,attr"`
	Device  DiskDeviceType `xml:"device,attr"`
	Source  DiskSource     `xml:"source"`
	Target  DiskTarget     `xml:"target"`
}

type GraphicsDeviceType string

const (
	GraphicsDeviceTypeVNC GraphicsDeviceType = "vnc"
)

type GraphicsDevice struct {
	XMLName xml.Name           `xml:"graphics"`
	Type    GraphicsDeviceType `xml:"type,attr"`
	Port    int                `xml:"port,attr"`
	KeyMap  string             `xml:"keymap,attr"`
}
