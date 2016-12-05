package builtin

import (
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"time"
)

const (
	cmdPing = "core.ping"
)

func init() {
	pm.CmdMap[cmdPing] = process.NewInternalProcessFactory(ping)
}

func ping(cmd *core.Command) (interface{}, error) {
	return fmt.Sprintf("PONG %s", time.Now()), nil
}
