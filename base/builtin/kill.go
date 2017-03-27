package builtin

import (
	"encoding/json"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"syscall"
)

const (
	cmdKill = "process.kill"
)

func init() {
	pm.CmdMap[cmdKill] = process.NewInternalProcessFactory(kill)
}

type killData struct {
	ID     string         `json:"id"`
	Signal syscall.Signal `json:"signal"`
}

func kill(cmd *core.Command) (interface{}, error) {
	//load data
	data := killData{}
	err := json.Unmarshal(*cmd.Arguments, &data)

	if err != nil {
		return nil, err
	}

	if data.Signal == syscall.Signal(0) {
		data.Signal = syscall.SIGTERM
	}

	runner, ok := pm.GetManager().Runners()[data.ID]
	if !ok {
		return false, nil
	}

	return true, runner.Process().Signal(data.Signal)

}
