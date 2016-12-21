package builtin

import (
	"encoding/json"
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

const (
	monitorDisk    = "disk"
	monitorCPU     = "cpu"
	monitorNetwork = "network"
	monitorMemory  = "memory"
)

type monitor struct{}

func init() {
	var m monitor
	pm.CmdMap["monitor"] = process.NewInternalProcessFactory(m.monitor)
}

func (m monitor) monitor(cmd *core.Command) (interface{}, error) {
	var args struct {
		Domain string `json:"domain"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	switch strings.ToLower(args.Domain) {
	case monitorDisk:
		return nil, m.disk()
	case monitorCPU:
		return nil, m.cpu()
	case monitorMemory:
		return nil, m.mem()
	default:
		return nil, fmt.Errorf("invalid monitoring domain: %s", args.Domain)
	}

	return nil, nil
}

func (m *monitor) disk() error {
	counters, err := disk.IOCounters()
	if err != nil {
		return err
	}

	p := pm.GetManager()
	for name, counter := range counters {
		key := fmt.Sprintf("%%s@phys.%s", name)

		p.Aggregate(pm.AggreagteDifference,
			fmt.Sprintf(key, "disk.iops.read"),
			float64(counter.ReadCount),
			"",
		)

		p.Aggregate(pm.AggreagteDifference,
			fmt.Sprintf(key, "disk.iops.write"),
			float64(counter.WriteCount),
			"",
		)

		p.Aggregate(pm.AggreagteDifference,
			fmt.Sprintf(key, "disk.throughput.read"),
			float64(counter.ReadBytes/1024),
			"",
		)

		p.Aggregate(pm.AggreagteDifference,
			fmt.Sprintf(key, "disk.iops.write"),
			float64(counter.WriteCount/1024),
			"",
		)
	}

	return nil
}

func (m *monitor) cpu() error {
	times, err := cpu.Times(true)
	if err != nil {
		return err
	}

	p := pm.GetManager()
	for nr, t := range times {
		key := fmt.Sprintf("%%s@pyhs.%d", nr)

		p.Aggregate(pm.AggreagteDifference,
			fmt.Sprintf(key, "machine.CPU.utilisation"),
			t.System+t.User,
			"",
		)
	}

	percent, err := cpu.Percent(time.Second, true)
	if err != nil {
		return err
	}

	for nr, v := range percent {
		key := fmt.Sprintf("%%s@pyhs.%d", nr)

		p.Aggregate(pm.AggreagteAverage,
			fmt.Sprintf(key, "machine.CPU.percent"),
			v,
			"",
		)
	}

	const StatFile = "/proc/stat"
	stat, err := ioutil.ReadFile(StatFile)
	if err != nil {
		return err
	}

	statmap := make(map[string]string)
	for _, line := range strings.Split(string(stat), "\n") {
		var key, value string
		if n, err := fmt.Sscanf(line, "%s %v", &key, &value); n == 2 && err == nil {
			statmap[key] = value
		}
	}

	if ctxt, ok := statmap["ctxt"]; ok {
		v, _ := strconv.ParseFloat(ctxt, 64)
		p.Aggregate(pm.AggreagteDifference,
			"machine.CPU.contextswitch@phys",
			v,
			"",
		)
	}

	if intr, ok := statmap["intr"]; ok {
		v, _ := strconv.ParseFloat(intr, 64)
		p.Aggregate(pm.AggreagteDifference,
			"machine.CPU.interrupts@phys",
			v,
			"",
		)
	}

	return nil
}

func (m *monitor) mem() error {
	return nil
}
