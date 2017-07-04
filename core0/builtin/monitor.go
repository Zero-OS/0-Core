package builtin

import (
	"encoding/json"
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
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
	m := (*monitor)(nil)
	pm.CmdMap["monitor"] = process.NewInternalProcessFactory(m.monitor)
}

func (m *monitor) monitor(cmd *core.Command) (interface{}, error) {
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
		return nil, m.memory()
	case monitorNetwork:
		return nil, m.network()
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
		p.Aggregate(pm.AggreagteDifference,
			"disk.iops.read",
			float64(counter.ReadCount),
			name, pm.Tag{"type", "phys"},
		)

		p.Aggregate(pm.AggreagteDifference,
			"disk.iops.write",
			float64(counter.WriteCount),
			name, pm.Tag{"type", "phys"},
		)

		p.Aggregate(pm.AggreagteDifference,
			"disk.throughput.read",
			float64(counter.ReadBytes/1024),
			name, pm.Tag{"type", "phys"},
		)

		p.Aggregate(pm.AggreagteDifference,
			"disk.throughput.write",
			float64(counter.WriteBytes/1024),
			name, pm.Tag{"type", "phys"},
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
		p.Aggregate(pm.AggreagteDifference,
			"machine.CPU.utilisation",
			t.System+t.User,
			fmt.Sprint(nr), pm.Tag{"type", "phys"},
		)
	}

	percent, err := cpu.Percent(time.Second, true)
	if err != nil {
		return err
	}

	for nr, v := range percent {
		p.Aggregate(pm.AggreagteAverage,
			"machine.CPU.percent",
			v,
			fmt.Sprint(nr), pm.Tag{"type", "phys"},
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
			"machine.CPU.contextswitch",
			v,
			"", pm.Tag{"type", "phys"},
		)
	}

	if intr, ok := statmap["intr"]; ok {
		v, _ := strconv.ParseFloat(intr, 64)
		p.Aggregate(pm.AggreagteDifference,
			"machine.CPU.interrupts",
			v,
			"", pm.Tag{"type", "phys"},
		)
	}

	return nil
}

func (m *monitor) memory() error {
	virt, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	p := pm.GetManager()

	p.Aggregate(pm.AggreagteAverage,
		"machine.memory.ram.available",
		float64(virt.Available)/(1024.*1024.),
		"", pm.Tag{"type", "phys"},
	)

	swap, err := mem.SwapMemory()
	if err != nil {
		return err
	}

	p.Aggregate(pm.AggreagteAverage,
		"machine.memory.swap.left",
		float64(swap.Free)/(1024.*1024.),
		"", pm.Tag{"type", "phys"},
	)

	p.Aggregate(pm.AggreagteAverage,
		"machine.memory.swap.used",
		float64(swap.Used)/(1024.*1024.),
		"", pm.Tag{"type", "phys"},
	)

	return nil
}

func (m *monitor) network() error {
	counters, err := net.IOCounters(true)
	if err != nil {
		return err
	}

	p := pm.GetManager()
	for _, counter := range counters {
		p.Aggregate(pm.AggreagteDifference,
			"network.throughput.outgoing",
			float64(counter.BytesSent)/(1024.*1024.),
			counter.Name, pm.Tag{"type", "phys"},
		)

		p.Aggregate(pm.AggreagteDifference,
			"network.throughput.incoming",
			float64(counter.BytesRecv)/(1024.*1024.),
			counter.Name, pm.Tag{"type", "phys"},
		)

		p.Aggregate(pm.AggreagteDifference,
			"network.packets.tx",
			float64(counter.PacketsSent)/(1024.*1024.),
			counter.Name, pm.Tag{"type", "phys"},
		)

		p.Aggregate(pm.AggreagteDifference,
			"network.packets.rx",
			float64(counter.PacketsRecv)/(1024.*1024.),
			counter.Name, pm.Tag{"type", "phys"},
		)
	}

	return nil
}
