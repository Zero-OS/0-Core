package options

import (
	"flag"
	"fmt"
	"os"
)

type AppOptions struct {
	version       bool
	coreID        uint64
	redisSocket   string
	redisPassword string
	replyTo       string
	maxJobs       int
	hostname      string
	unprivileged  bool
}

func (o *AppOptions) Version() bool {
	return o.version
}

func (o *AppOptions) CoreID() uint64 {
	return o.coreID
}

func (o *AppOptions) RedisSocket() string {
	return o.redisSocket
}

func (o *AppOptions) RedisPassword() string {
	return o.redisPassword
}

func (o *AppOptions) ReplyTo() string {
	return o.replyTo
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
	if o.coreID == 0 {
		errors = append(errors, fmt.Errorf("-core-id is required"))
	}

	if o.redisSocket == "" {
		errors = append(errors, fmt.Errorf("-redis-socket is required"))
	}

	return errors
}

var Options AppOptions

func init() {
	help := false
	flag.BoolVar(&help, "h", false, "Print this help screen")
	flag.BoolVar(&Options.version, "v", false, "Print the version and exits")
	flag.Uint64Var(&Options.coreID, "core-id", 0, "Core ID")
	flag.StringVar(&Options.redisSocket, "redis-socket", "", "Path to the redis socket")
	flag.StringVar(&Options.redisPassword, "redis-password", "", "Redis password [optional]")
	flag.StringVar(&Options.replyTo, "reply-to", "corex:results", "Reply to queue")
	flag.IntVar(&Options.maxJobs, "max-jobs", 100, "Max number of jobs that can run concurrently")
	flag.StringVar(&Options.hostname, "hostname", "", "Hostname of the container")
	flag.BoolVar(&Options.unprivileged, "unprivileged", false, "Unprivileged container (strips down container capabilites)")

	flag.Parse()

	if Options.hostname == "" {
		Options.hostname = fmt.Sprintf("core-%d", Options.coreID)
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
