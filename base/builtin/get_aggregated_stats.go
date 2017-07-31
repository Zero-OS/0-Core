package builtin

import (
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	psutil "github.com/shirou/gopsutil/process"
	"os"
)

const (
	cmdGetAggregatedStats = "core.state"
)

type aggregatedStatsMgr struct {
	agent *psutil.Process
}

func init() {
	agent, err := psutil.NewProcess(int32(os.Getpid()))
	if err != nil {
		log.Errorf("Failed to get reference to agent process: %s", err)
	}

	mgr := &aggregatedStatsMgr{
		agent: agent,
	}

	pm.CmdMap[cmdGetAggregatedStats] = process.NewInternalProcessFactory(mgr.getAggregatedStats)
}

func (mgr *aggregatedStatsMgr) getAggregatedStats(cmd *core.Command) (interface{}, error) {
	stat := process.ProcessStats{}

	for _, runner := range pm.Jobs() {
		ps := runner.Process()
		if ps == nil {
			continue
		}
		stats, ok := ps.(process.Stater)
		if !ok {
			continue
		}

		processStats := stats.Stats()
		stat.CPU += processStats.CPU
		stat.RSS += processStats.RSS
		stat.Swap += processStats.Swap
		stat.VMS += processStats.VMS
	}

	//also get agent cpu and memory consumption.
	if mgr.agent != nil {
		agentCPU, err := mgr.agent.Percent(0)
		if err == nil {
			stat.CPU += agentCPU
		} else {
			log.Errorf("%s", err)
		}

		agentMem, err := mgr.agent.MemoryInfo()
		if err == nil {
			stat.RSS += agentMem.RSS
			stat.Swap += agentMem.Swap
			stat.VMS += agentMem.VMS
		} else {
			log.Errorf("%s", err)
		}
	}

	return stat, nil
}
