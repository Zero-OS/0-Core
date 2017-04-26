package logger

import (
	"encoding/json"
	"fmt"

	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	logging "github.com/op/go-logging"
)

type logMgr struct{}

func init() {
	l := (*logMgr)(nil)
	pm.CmdMap["logger.setlevel"] = process.NewInternalProcessFactory(l.setlevel)
}

type LogLevel struct {
	Level string `json:"level"`
}

func (l *logMgr) setlevel(cmd *core.Command) (interface{}, error) {
	var args LogLevel

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	level, err := logging.LogLevel(args.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %s", args.Level)
	}

	logging.SetLevel(level, "")

	return nil, nil

}
