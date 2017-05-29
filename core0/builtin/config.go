package builtin

import (
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"github.com/zero-os/0-core/base/settings"
)

type configMgr struct{}

func init() {
	c := (*configMgr)(nil)
	pm.CmdMap["config.get"] = process.NewInternalProcessFactory(c.get)
}

func (c *configMgr) get(cmd *core.Command) (interface{}, error) {
	return settings.Settings, nil
}
