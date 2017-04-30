package builtin

import (
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"io/ioutil"
	"strconv"
	"strings"
)

const (
	cmdGetCPUInfo  = "info.cpu"
	cmdGetDiskInfo = "info.disk"
	cmdGetMemInfo  = "info.mem"
	cmdGetNicInfo  = "info.nic"
	cmdGetOsInfo   = "info.os"
)

func init() {
	pm.CmdMap[cmdGetCPUInfo] = process.NewInternalProcessFactory(getCPUInfo)
	pm.CmdMap[cmdGetDiskInfo] = process.NewInternalProcessFactory(getDiskInfo)
	pm.CmdMap[cmdGetMemInfo] = process.NewInternalProcessFactory(getMemInfo)
	pm.CmdMap[cmdGetNicInfo] = process.NewInternalProcessFactory(getNicInfo)
	pm.CmdMap[cmdGetOsInfo] = process.NewInternalProcessFactory(getOsInfo)
}

func getCPUInfo(cmd *core.Command) (interface{}, error) {
	return cpu.Info()
}

func getDiskInfo(cmd *core.Command) (interface{}, error) {
	return disk.Partitions(true)
}

func getMemInfo(cmd *core.Command) (interface{}, error) {
	return mem.VirtualMemory()
}

type NicInfo struct {
	net.InterfaceStat
	Speed uint32 `json:"speed"`
}

func getNicInfo(cmd *core.Command) (interface{}, error) {
	var speed uint32
	ifcs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	ret := make([]NicInfo, len(ifcs))
	for i, ifc := range ifcs {
		ret[i].MTU = ifc.MTU
		ret[i].Name = ifc.Name
		ret[i].HardwareAddr = ifc.HardwareAddr
		ret[i].Flags = ifc.Flags
		ret[i].Addrs = ifc.Addrs
		dat, err := ioutil.ReadFile("/sys/class/net/" + ifc.Name + "/speed")
		if err != nil {
			speed = 0
		} else {
			speedint, err := strconv.Atoi(strings.Trim(string(dat), "\n"))
			if err != nil {
				speed = 0
			} else {
				speed = uint32(speedint)
			}
		}
		ret[i].Speed = speed
	}
	return ret, nil
}

func getOsInfo(cmd *core.Command) (interface{}, error) {
	return host.Info()
}
