package bootstrap

import (
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/g8os/core0/base/utils"
)

const (
	PluginSearchPath = "/var/lib/corex/plugins"
	ManifestSymbol   = "Manifest"
	PluginSymbol     = "Plugin"
	PluginExt        = ".so"
)

type Plugin struct {
	Path    string
	Exports []string
}

type PluginsSettings struct {
	Plugin map[string]Plugin
}

func (b *Bootstrap) pluginFactory(path string, fn string) process.ProcessFactory {
	return func(table process.PIDTable, srcCmd *core.Command) process.Process {
		cmd := &core.Command{
			ID:      srcCmd.ID,
			Command: process.CommandSystem,
			Arguments: core.MustArguments(process.SystemCommandArguments{
				Name: path,
				Args: []string{fn, string(*srcCmd.Arguments)},
			}),
			Queue:           srcCmd.Queue,
			StatsInterval:   srcCmd.StatsInterval,
			MaxTime:         srcCmd.MaxTime,
			MaxRestart:      srcCmd.MaxRestart,
			RecurringPeriod: srcCmd.RecurringPeriod,
			LogLevels:       srcCmd.LogLevels,
			Tags:            srcCmd.Tags,
		}

		return process.NewSystemProcess(table, cmd)
	}
}

func (b *Bootstrap) plugin(domain string, plugin Plugin) {
	for _, export := range plugin.Exports {
		cmd := fmt.Sprintf("%s.%s", domain, export)

		pm.CmdMap[cmd] = b.pluginFactory(plugin.Path, export)
	}
}

func (b *Bootstrap) plugins() error {
	var plugins PluginsSettings
	if err := utils.LoadTomlFile("/.plugin.toml", &plugins); err != nil {
		return err
	}

	for domain, plugin := range plugins.Plugin {
		b.plugin(domain, plugin)
	}

	return nil
}
