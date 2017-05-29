package builtin

import (
	"fmt"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"gopkg.in/bufio.v1"
	"io/ioutil"
	gonet "net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	cmdGetCPUInfo  = "info.cpu"
	cmdGetDiskInfo = "info.disk"
	cmdGetMemInfo  = "info.mem"
	cmdGetNicInfo  = "info.nic"
	cmdGetOsInfo   = "info.os"
	cmdGetPortInfo = "info.port"
)

func init() {
	pm.CmdMap[cmdGetCPUInfo] = process.NewInternalProcessFactory(getCPUInfo)
	pm.CmdMap[cmdGetDiskInfo] = process.NewInternalProcessFactory(getDiskInfo)
	pm.CmdMap[cmdGetMemInfo] = process.NewInternalProcessFactory(getMemInfo)
	pm.CmdMap[cmdGetNicInfo] = process.NewInternalProcessFactory(getNicInfo)
	pm.CmdMap[cmdGetOsInfo] = process.NewInternalProcessFactory(getOsInfo)
	pm.CmdMap[cmdGetPortInfo] = process.NewInternalProcessFactory(getPortInfo)
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

type Port struct {
	Network string   `json:"network"`
	Port    uint16   `json:"port,omitempty"`
	Unix    string   `json:"unix,omitempty"`
	IP      gonet.IP `json:"ip,omitempty"`
	PID     uint64   `json:"pid"`

	inode uint64
}

func parseIP(s string) (ip gonet.IP) {
	if _, err := fmt.Sscanf(s, "%x", &ip); err != nil {
		return
	}
	//network to host byte order for generic ip4 and ip6
	for i := 0; i < len(ip); i += 4 {
		for j := 0; j < 2; j++ {
			ip[i+j], ip[i+3-j] = ip[i+3-j], ip[i+j]
		}
	}
	return
}

func getTCPUDPInfo() ([]*Port, error) {
	var ports []*Port
	for _, network := range []string{"tcp", "tcp6", "udp", "udp6"} {
		p := path.Join("/proc", "net", network)
		content, err := ioutil.ReadFile(p)
		if err != nil {
			log.Debugf("failed to read %s", p)
			continue
		}
		buf := bufio.NewBuffer(content)
		for line, err := buf.ReadString('\n'); err == nil; line, err = buf.ReadString('\n') {
			fields := strings.Fields(line)
			if len(fields) < 4 || fields[1] == "local_address" {
				continue
			}
			local := fields[1]
			mode := fields[3]
			if !(mode == "0A" || mode == "07") {
				//not listening
				continue
			}
			localParts := strings.Split(local, ":")
			port, err := strconv.ParseUint(localParts[1], 16, 16)
			if err != nil {
				return nil, err
			}

			inode, _ := strconv.ParseUint(fields[9], 10, 64)

			ports = append(ports, &Port{
				Network: network,
				Port:    uint16(port),
				IP:      parseIP(localParts[0]),
				inode:   inode,
			})
		}
	}

	return ports, nil
}

func getUnixSocketInfo() ([]*Port, error) {
	var ports []*Port
	p := path.Join("/proc", "net", "unix")
	content, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}
	buf := bufio.NewBuffer(content)
	for line, err := buf.ReadString('\n'); err == nil; line, err = buf.ReadString('\n') {
		fields := strings.Fields(line)
		if len(fields) < 8 || fields[0] == "Num" {
			continue
		}
		state := fields[5]
		if state != "01" {
			continue
		}

		inode, _ := strconv.ParseUint(fields[6], 10, 64)
		unix := fields[7]

		ports = append(ports, &Port{
			Network: "unix",
			Unix:    path.Clean(unix),
			inode:   inode,
		})
	}

	return ports, nil
}

func getProcessSocketsInodes(pid uint64, m map[uint64]uint64) {
	base := fmt.Sprintf("/proc/%d/fd", pid)
	links, err := ioutil.ReadDir(base)
	if err != nil {
		//possibility process is gone before we able to read the fd links
		log.Debugf("failed to readdir %s:%s", base, err)
		return
	}

	for _, link := range links {
		lp := path.Join(base, link.Name())
		target, err := os.Readlink(lp)
		if err != nil {
			log.Debugf("failed to readlink %s: %s", lp, err)
			continue
		}
		var inode uint64
		if _, err := fmt.Sscanf(target, "socket:[%d]", &inode); err == nil {
			m[inode] = pid
		}
	}
}

func getSocketsInodes() map[uint64]uint64 {
	m := make(map[uint64]uint64)

	wk := func(path string, info os.FileInfo, err error) error {
		if path == "/proc" {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		pid, err := strconv.ParseUint(info.Name(), 10, 64)
		if err != nil {
			return filepath.SkipDir
		}

		getProcessSocketsInodes(pid, m)
		return filepath.SkipDir
	}

	filepath.Walk("/proc", wk)
	return m
}

func getPortInfo(cmd *core.Command) (interface{}, error) {
	ports, err := getTCPUDPInfo()
	if err != nil {
		return nil, err
	}
	unix, err := getUnixSocketInfo()
	ports = append(ports, unix...)

	inodes := getSocketsInodes()
	log.Debugf("Inodes: %v", inodes)
	for _, port := range ports {
		pid, ok := inodes[port.inode]
		if ok {
			port.PID = pid
		}
	}

	return ports, nil
}
