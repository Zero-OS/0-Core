package builtin

import (
	"encoding/json"
	"fmt"

	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"github.com/op/go-logging"
	"os"
	"syscall"
)

type logMgr struct{}

func init() {
	l := (*logMgr)(nil)
	pm.CmdMap["logger.set_level"] = process.NewInternalProcessFactory(l.setLevel)
	pm.CmdMap["logger.reopen"] = process.NewInternalProcessFactory(l.reopen)
}

type LogLevel struct {
	Level string `json:"level"`
}

func (l *logMgr) setLevel(cmd *core.Command) (interface{}, error) {
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

func (l *logMgr) reopen(cmd *core.Command) (interface{}, error) {
	return nil, syscall.Kill(os.Getpid(), syscall.SIGUSR1)
}
