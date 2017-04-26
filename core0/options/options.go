package options

import (
	"flag"
	"fmt"
	"os"
)

type AppOptions struct {
	cfg     string
	version bool
	Kernel  KernelOptions
}

func (o *AppOptions) Config() string {
	return o.cfg
}

func (o *AppOptions) Version() bool {
	return o.version
}

var Options AppOptions

func init() {
	help := false
	flag.BoolVar(&help, "h", false, "Print this help screen")
	flag.StringVar(&Options.cfg, "c", "/etc/g8os/g8os.toml", "Path to config file")
	flag.BoolVar(&Options.version, "v", false, "Prints version and exit")
	flag.Parse()

	printHelp := func() {
		fmt.Println("core [options]")
		flag.PrintDefaults()
	}

	if help {
		printHelp()
		os.Exit(0)
	}
	Options.Kernel = getKernelParams()
}
