package builtin

import (
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/g8os/core0/base/settings"
)

type configMgr struct{}

func init() {
	c := (*configMgr)(nil)
	pm.CmdMap["config.get"] = process.NewInternalProcessFactory(c.get)
}

func (c *configMgr) get(cmd *core.Command) (interface{}, error) {
	return settings.Settings, nil
}
