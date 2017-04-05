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
	Queue   bool
	Exports []string
}

type PluginsSettings struct {
	Plugin map[string]Plugin
}

func (b *Bootstrap) pluginFactory(domain string, plugin *Plugin, fn string) process.ProcessFactory {
	return func(table process.PIDTable, srcCmd *core.Command) process.Process {
		queue := srcCmd.Queue
		if plugin.Queue {
			queue = domain
		}
		cmd := &core.Command{
			ID:      srcCmd.ID,
			Command: process.CommandSystem,
			Arguments: core.MustArguments(process.SystemCommandArguments{
				Name: plugin.Path,
				Args: []string{fn, string(*srcCmd.Arguments)},
			}),
			Queue:           queue,
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

		pm.CmdMap[cmd] = b.pluginFactory(domain, &plugin, export)
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
