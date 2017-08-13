package options

import (
	"flag"
	"fmt"
	"github.com/zero-os/0-core/base/utils"
	"os"
)

type AppOptions struct {
	cfg     string
	version bool
	agent   bool
	Kernel  utils.KernelOptions
}

func (o *AppOptions) Config() string {
	return o.cfg
}

func (o *AppOptions) Agent() bool {
	return o.agent
}

func (o *AppOptions) Version() bool {
	return o.version
}

var Options AppOptions

func init() {
	help := false
	flag.BoolVar(&help, "h", false, "Print this help screen")
	flag.StringVar(&Options.cfg, "c", "/etc/g8os/zero-os.toml", "Path to config file")
	flag.BoolVar(&Options.version, "v", false, "Prints version and exit")
	flag.BoolVar(&Options.agent, "a", false, "Run in agent mode (not init)")
	flag.Parse()

	printHelp := func() {
		fmt.Println("core [options]")
		flag.PrintDefaults()
	}

	if help {
		printHelp()
		os.Exit(0)
	}

	Options.Kernel = utils.GetKernelOptions()
}
