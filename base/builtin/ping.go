package builtin

import (
	"fmt"
	base "github.com/zero-os/0-core/base"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
)

const (
	cmdPing = "core.ping"
)

func init() {
	pm.CmdMap[cmdPing] = process.NewInternalProcessFactory(ping)
}

func ping(cmd *core.Command) (interface{}, error) {
	return fmt.Sprintf("PONG %s", base.Version()), nil
}
