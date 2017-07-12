package bootstrap

import (
	"fmt"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"github.com/zero-os/0-core/base/utils"
	"strings"
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

func (b *Bootstrap) pluginFactory(plugin *Plugin, fn string) process.ProcessFactory {
	return func(table process.PIDTable, srcCmd *core.Command) process.Process {
		cmd := &core.Command{
			ID: srcCmd.ID,
			Arguments: core.MustArguments(process.SystemCommandArguments{
				Name: plugin.Path,
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
	if plugin.Queue {
		//if plugin requires queuing we make sure when a command is pushed (from a cient)
		//that we force a queue on it.
		pm.GetManager().AddPreProcessor(func(cmd *core.Command) {
			if strings.HasPrefix(cmd.Command, domain+".") {
				log.Debugf("setting command queue to: %s", domain)
				cmd.Queue = domain
			}
		})
	}

	for _, export := range plugin.Exports {
		cmd := fmt.Sprintf("%s.%s", domain, export)
		pm.CmdMap[cmd] = b.pluginFactory(&plugin, export)
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
