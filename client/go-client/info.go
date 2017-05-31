package client

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

type InfoManager interface {
	CPU() ([]cpu.InfoStat, error)
	Mem() (mem.VirtualMemoryStat, error)
	Nic() ([]net.InterfaceStat, error)
	Disk() ([]disk.PartitionStat, error)
	OS() (host.InfoStat, error)
}

type infoMgr struct {
	Client
}

func Info(cl Client) InfoManager {
	return &infoMgr{cl}
}

func (i *infoMgr) CPU() ([]cpu.InfoStat, error) {
	res, err := sync(i, "info.cpu", A{})
	if err != nil {
		return nil, err
	}

	var info []cpu.InfoStat
	return info, res.Json(&info)
}

func (i *infoMgr) Mem() (mem.VirtualMemoryStat, error) {
	var m mem.VirtualMemoryStat
	res, err := sync(i, "info.mem", A{})
	if err != nil {
		return m, err
	}

	if err := res.Json(&m); err != nil {
		return m, err
	}

	return m, nil
}

func (i *infoMgr) Nic() ([]net.InterfaceStat, error) {
	res, err := sync(i, "info.nic", A{})
	if err != nil {
		return nil, err
	}

	var info []net.InterfaceStat
	return info, res.Json(&info)
}

func (i *infoMgr) Disk() ([]disk.PartitionStat, error) {
	res, err := sync(i, "info.disk", A{})
	if err != nil {
		return nil, err
	}

	var info []disk.PartitionStat
	return info, res.Json(&info)
}

func (i *infoMgr) OS() (host.InfoStat, error) {
	var m host.InfoStat
	res, err := sync(i, "info.os", A{})
	if err != nil {
		return m, err
	}

	if err := res.Json(&m); err != nil {
		return m, err
	}

	return m, nil
}
