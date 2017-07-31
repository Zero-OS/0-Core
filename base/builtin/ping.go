package builtin

import (
	"fmt"
	base "github.com/zero-os/0-core/base"
	"github.com/zero-os/0-core/base/pm"
)

const (
	cmdPing = "core.ping"
)

func init() {
	pm.RegisterBuiltIn(cmdPing, ping)
}

func ping(cmd *pm.Command) (interface{}, error) {
	return fmt.Sprintf("PONG %s", base.Version()), nil
}
