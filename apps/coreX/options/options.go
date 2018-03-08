package options

import (
	"flag"
	"fmt"
	"os"
)

type AppOptions struct {
	version      bool
	maxJobs      int
	hostname     string
	unprivileged bool
}

func (o *AppOptions) Version() bool {
	return o.version
}

func (o *AppOptions) MaxJobs() int {
	return o.maxJobs
}

func (o *AppOptions) Hostname() string {
	return o.hostname
}

func (o *AppOptions) Unprivileged() bool {
	return o.unprivileged
}

func (o *AppOptions) Validate() []error {
	errors := make([]error, 0)

	return errors
}

var Options AppOptions

func init() {
	help := false
	flag.BoolVar(&help, "h", false, "Print this help screen")
	flag.BoolVar(&Options.version, "v", false, "Print the version and exits")
	flag.IntVar(&Options.maxJobs, "max-jobs", 100000, "Max number of jobs that can run concurrently")
	flag.StringVar(&Options.hostname, "hostname", "", "Hostname of the container")
	flag.BoolVar(&Options.unprivileged, "unprivileged", false, "Unprivileged container (strips down container capabilites)")

	flag.Parse()

	if Options.hostname == "" {
		Options.hostname = "corex"
	}

	printHelp := func() {
		fmt.Println("coreX [options]")
		flag.PrintDefaults()
	}

	if help {
		printHelp()
		os.Exit(0)
	}
}
