package builtin

import (
	"github.com/Jumpscale/agent8/agent/lib/pm"
	"github.com/Jumpscale/agent8/agent/lib/pm/core"
	"github.com/Jumpscale/agent8/agent/lib/pm/process"
)

const (
	cmdPing = "ping"
)

func init() {
	pm.CmdMap[cmdPing] = process.NewInternalProcessFactory(ping)
}

func ping(cmd *core.Cmd) (interface{}, error) {
	return "pong", nil
}
