package bootstrap

import (
	"fmt"
	pl "github.com/g8os/core0/base/plugin"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/process"
	"os"
	"path"
	"path/filepath"
	"plugin"
	"strings"
)

const (
	PluginSearchPath = "/var/lib/corex/plugins"
	PluginSymbol     = "Plugin"
	PluginExt        = ".so"
)

func (b *Bootstrap) plugin(name string) error {
	plgn, err := plugin.Open(name)
	if err != nil {
		return err
	}

	entry, err := plgn.Lookup(PluginSymbol)
	if err != nil {
		return err
	}

	if entry, ok := entry.(*pl.Plugin); ok {
		domain := entry.Domain
		if domain == "" {
			d := path.Base(name)
			domain = strings.TrimSuffix(d, PluginExt)
		}

		for name, fn := range entry.Commands {
			pm.CmdMap[fmt.Sprintf("%s.%s", domain, name)] = process.NewInternalProcessFactory(fn)
		}
	} else {
		return fmt.Errorf("not a comptaible plugin")
	}

	return nil
}

func (b *Bootstrap) plugins() error {
	walk := func(name string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		ext := path.Ext(name)
		if ext != PluginExt {
			return nil
		}

		if err := b.plugin(name); err != nil {
			log.Errorf("failed to load plugin (%s): %s", name, err)
		}

		return nil
	}

	return filepath.Walk(PluginSearchPath, walk)
}
