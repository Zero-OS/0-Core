package builtin

import (
	"encoding/json"
	"fmt"

	"github.com/op/go-logging"
	"github.com/zero-os/0-core/base/pm"
	"os"
	"syscall"
)

type logMgr struct{}

func init() {
	l := (*logMgr)(nil)
	pm.RegisterBuiltIn("logger.set_level", l.setLevel)
	pm.RegisterBuiltIn("logger.reopen", l.reopen)
}

type LogLevel struct {
	Level string `json:"level"`
}

func (l *logMgr) setLevel(cmd *pm.Command) (interface{}, error) {
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

func (l *logMgr) reopen(cmd *pm.Command) (interface{}, error) {
	return nil, syscall.Kill(os.Getpid(), syscall.SIGUSR1)
}
