package builtin

import (
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/shirou/gopsutil/net"
)

const (
	cmdGetNicInfo = "info.nic"
)

func init() {
	pm.CmdMap[cmdGetNicInfo] = process.NewInternalProcessFactory(getNicInfo)
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
