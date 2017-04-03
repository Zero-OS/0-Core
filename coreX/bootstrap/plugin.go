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
	ManifestSymbol   = "Manifest"
	PluginSymbol     = "Plugin"
	PluginExt        = ".so"
)

func (b *Bootstrap) pluginV1(domain string, p *plugin.Plugin) error {
	sym, err := p.Lookup(PluginSymbol)
	if err != nil {
		return err
	}

	commands, ok := sym.(*pl.Commands)

	if !ok {
		return fmt.Errorf("plugin(v1) wrong plugin object")
	}

	for name, fn := range *commands {
		pm.CmdMap[fmt.Sprintf("%s.%s", domain, name)] = process.NewInternalProcessFactory(fn)
	}

	return nil
}

func (b *Bootstrap) plugin(name string) error {
	plgn, err := plugin.Open(name)
	if err != nil {
		return err
	}

	sym, err := plgn.Lookup(ManifestSymbol)
	if err != nil {
		return err
	}

	man, ok := sym.(*pl.Manifest)
	if !ok {
		return fmt.Errorf("not a comptaible plugin")
	}

	domain := man.Domain
	if domain == "" {
		d := path.Base(name)
		domain = strings.TrimSuffix(d, PluginExt)
	}

	switch man.Version {
	case pl.Version_1:
		fallthrough
	default:
		return b.pluginV1(domain, plgn)
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
