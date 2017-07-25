package builtin

import (
	"encoding/json"
	"fmt"
	"github.com/shirou/gopsutil/process"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	ps "github.com/zero-os/0-core/base/pm/process"
	"io/ioutil"
	"strings"
	"syscall"
)

const (
	cmdProcessList = "process.list"
	cmdProcessKill = "process.kill"
)

func init() {
	pm.CmdMap[cmdProcessList] = ps.NewInternalProcessFactory(processList)
	pm.CmdMap[cmdProcessKill] = ps.NewInternalProcessFactory(processKill)
}

type processListArguments struct {
	PID int32 `json:"pid"`
}

type Process struct {
	PID        int32                `json:"pid"`
	PPID       int32                `json:"ppid"`
	Cmdline    string               `json:"cmdline"`
	Name       string               `json:"name"`
	CreateTime int64                `json:"createtime"`
	Cpu        CPUStats             `json:"cpu"`
	RSS        uint64               `json:"rss"`
	VMS        uint64               `json:"vms"`
	Swap       uint64               `json:"swap"`
	RLimit     []process.RlimitStat `json:"rlimit"`
	OFD        int32                `json:"ofd"`
}

type CPUStats struct {
	GuestNice float64 `json:"guestnice"`
	Idle      float64 `json:"idle"`
	IoWait    float64 `json:"iowait"`
	Irq       float64 `json:"irq"`
	Nice      float64 `json:"nice"`
	SoftIrq   float64 `json:"softirq"`
	Steal     float64 `json:"steal"`
	Stolen    float64 `json:"stolen"`
	System    float64 `json:"system"`
	User      float64 `json:"user"`
}

func getProcessInfo(ps *process.Process) *Process {
	res := &Process{
		PID: ps.Pid,
	}

	//get PPID we don't use psutil for that because they actually use exec to get it which shouldn't be done outside
	//of the process manager
	if data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", ps.Pid)); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 4 {
			fmt.Sscanf(fields[3], "%d", &res.PPID)
		}
	}
	if cmd, err := ps.Cmdline(); err == nil {
		res.Cmdline = cmd
	}
	if name, err := ps.Name(); err == nil {
		res.Name = name
	}
	//
	if ct, err := ps.CreateTime(); err == nil {
		res.CreateTime = ct
	}

	if mem, err := ps.MemoryInfo(); err == nil {
		res.RSS = mem.RSS
		res.VMS = mem.VMS
		res.Swap = mem.Swap
	}

	if cpu, err := ps.Times(); err == nil {
		res.Cpu = CPUStats{
			GuestNice: cpu.GuestNice,
			Idle:      cpu.Idle,
			IoWait:    cpu.Iowait,
			Irq:       cpu.Irq,
			Nice:      cpu.Nice,
			SoftIrq:   cpu.Softirq,
			Steal:     cpu.Steal,
			Stolen:    cpu.Stolen,
			System:    cpu.System,
			User:      cpu.User,
		}
	}

	if ofd, err := ps.NumFDs(); err == nil {
		res.OFD = ofd
	}

	if rlimit, err := ps.Rlimit(); err == nil {
		res.RLimit = rlimit
	} else {
		res.RLimit = make([]process.RlimitStat, 0)
	}

	return res
}

func processList(cmd *core.Command) (interface{}, error) {
	var args processListArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	var pids []int32
	if args.PID > 0 {
		pids = []int32{args.PID}
	} else {
		var err error
		pids, err = process.Pids()
		if err != nil {
			return nil, err
		}
	}

	results := make([]*Process, 0, len(pids))
	for _, pid := range pids {
		ps, err := process.NewProcess(pid)
		if err != nil {
			//process pid gone before we read it
			continue
		}

		results = append(results, getProcessInfo(ps))
	}

	return results, nil
}

type processKillArguments struct {
	processListArguments
	Signal int `json:"signal"`
}

func processKill(cmd *core.Command) (interface{}, error) {
	var args processKillArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	if args.PID <= 1 {
		return nil, fmt.Errorf("invalid PID")
	}

	return nil, syscall.Kill(int(args.PID), syscall.Signal(args.Signal))
}
