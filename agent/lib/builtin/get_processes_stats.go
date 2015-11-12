package builtin

import (
	"encoding/json"
	"github.com/Jumpscale/agent2/agent/lib/pm"
	"github.com/Jumpscale/agent2/agent/lib/pm/core"
	"github.com/Jumpscale/agent2/agent/lib/pm/process"
)

const (
	cmdGetProcessesStats = "get_processes_stats"
)

func init() {
	pm.CmdMap[cmdGetProcessesStats] = process.NewInternalProcessFactory(getProcessesStats)
}

type getStatsData struct {
	Domain string `json:"domain"`
	Name   string `json:"name"`
}

func getProcessesStats(cmd *core.Cmd) (interface{}, error) {
	//load data
	data := getStatsData{}
	err := json.Unmarshal([]byte(cmd.Data), &data)
	if err != nil {
		return nil, err
	}

	stats := make([]*process.ProcessStats, 0, len(pm.GetManager().Processes()))

	for _, process := range pm.GetManager().Processes() {
		cmd := process.Cmd()

		if data.Domain != "" {
			if data.Domain != cmd.Args.GetString("domain") {
				continue
			}
		}

		if data.Name != "" {
			if data.Name != cmd.Args.GetString("name") {
				continue
			}
		}

		stats = append(stats, process.GetStats())
	}

	return stats, nil
}
