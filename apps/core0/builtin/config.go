package builtin

import (
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/settings"
)

type configMgr struct{}

func init() {
	c := (*configMgr)(nil)
	pm.RegisterBuiltIn("config.get", c.get)
}

func (c *configMgr) get(cmd *pm.Command) (interface{}, error) {
	return settings.Settings, nil
}
