package builtin

import (
	"encoding/json"
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
)

const (
	cmdGetProcessStats = "process.list"
)

func init() {
	pm.CmdMap[cmdGetProcessStats] = process.NewInternalProcessFactory(getProcessStats)
}

type getProcessStatsData struct {
	ID string `json:"id"`
}

type processData struct {
	process.ProcessStats
	StartTime int64         `json:"starttime"`
	Cmd       *core.Command `json:"cmd,omitempty"`
}

func getProcessStats(cmd *core.Command) (interface{}, error) {
	//load data
	data := getProcessStatsData{}
	err := json.Unmarshal(*cmd.Arguments, &data)
	if err != nil {
		return nil, err
	}

	stats := make([]processData, 0, len(pm.GetManager().Runners()))
	var runners []pm.Runner

	if data.ID != "" {
		runner, ok := pm.GetManager().Runners()[data.ID]

		if !ok {
			return nil, fmt.Errorf("Process with id '%s' doesn't exist", data.ID)
		}

		runners = []pm.Runner{runner}
	} else {
		for _, runner := range pm.GetManager().Runners() {
			runners = append(runners, runner)
		}
	}

	for _, runner := range runners {
		s := processData{
			Cmd:       runner.Command(),
			StartTime: runner.StartTime(),
		}

		ps := runner.Process()

		if stater, ok := ps.(process.Stater); ok {
			psStat := stater.Stats()
			s.CPU = psStat.CPU
			s.RSS = psStat.RSS
			s.VMS = psStat.VMS
			s.Swap = psStat.Swap
		}

		stats = append(stats, s)
	}

	return stats, nil
}
