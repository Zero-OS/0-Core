package builtin

import (
	"encoding/json"
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"syscall"
)

const (
	cmdJobList    = "job.list"
	cmdJobKill    = "job.kill"
	cmdJobKillAll = "job.killall"
)

func init() {
	pm.CmdMap[cmdJobList] = process.NewInternalProcessFactory(jobList)
	pm.CmdMap[cmdJobKill] = process.NewInternalProcessFactory(jobKill)
	pm.CmdMap[cmdJobKillAll] = process.NewInternalProcessFactory(jobKillAll)
}

type jobListArguments struct {
	ID string `json:"id"`
}

type processData struct {
	process.ProcessStats
	StartTime int64         `json:"starttime"`
	Cmd       *core.Command `json:"cmd,omitempty"`
}

func jobList(cmd *core.Command) (interface{}, error) {
	//load data
	var data jobListArguments
	err := json.Unmarshal(*cmd.Arguments, &data)
	if err != nil {
		return nil, err
	}

	var stats []processData
	var runners []pm.Runner

	if data.ID != "" {
		runner, ok := pm.GetManager().Runner(data.ID)

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

type jobKillArguments struct {
	ID     string         `json:"id"`
	Signal syscall.Signal `json:"signal"`
}

func jobKill(cmd *core.Command) (interface{}, error) {
	//load data
	data := jobKillArguments{}
	err := json.Unmarshal(*cmd.Arguments, &data)

	if err != nil {
		return nil, err
	}

	if data.Signal == syscall.Signal(0) {
		data.Signal = syscall.SIGTERM
	}

	runner, ok := pm.GetManager().Runner(data.ID)
	if !ok {
		return false, nil
	}

	if ps, ok := runner.Process().(process.Signaler); ok {
		if err := ps.Signal(data.Signal); err != nil {
			return false, err
		}
	}

	if data.Signal == syscall.SIGTERM || data.Signal == syscall.SIGKILL {
		runner.Terminate()
	}

	return true, nil

}

func jobKillAll(cmd *core.Command) (interface{}, error) {
	pm.GetManager().Killall()
	return true, nil
}
