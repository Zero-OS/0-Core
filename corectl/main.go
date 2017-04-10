package main

import (
	"github.com/codegangsta/cli"
	"github.com/op/go-logging"
	"os"
)

var (
	log = logging.MustGetLogger("corectl")
)

func init() {
	formatter := logging.MustStringFormatter("%{color}%{message}%{color:reset}")
	logging.SetFormatter(formatter)
}

func main() {
	app := cli.NewApp()
	app.Name = "corectl"
	app.Usage = "manage g8os"
	app.UsageText = "Query or send commands to g8os manager"
	app.Version = "1.0"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "socket, s",
			Value: "/var/run/core.sock",
			Usage: "Path to core socket",
		},
		cli.IntFlag{
			Name:  "timeout, t",
			Value: 0,
			Usage: "Commands that takes longer than this will get killed",
		},
		cli.BoolFlag{
			Name:  "async",
			Usage: "Run command asyncthronuslly (only commands that supports this)",
		},
		cli.StringFlag{
			Name:  "id",
			Usage: "Speicify porcess id, if not given a random guid will be generated",
		},
		cli.StringFlag{
			Name:  "container",
			Usage: "Container numeric ID or comma seperated list with tags (only with execute)",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "ping",
			Usage:  "checks connectivity with g8os",
			Action: WithTransport(ping),
		},
		{
			Name:            "execute",
			Usage:           "execute arbitary commands",
			Description:     "execute any command inside the core context",
			ArgsUsage:       "command [args]",
			Action:          WithTransport(system),
			SkipFlagParsing: true,
		},
		{
			Name:      "stop",
			Usage:     "stops a process with `id`",
			ArgsUsage: "id",
			Action:    WithTransport(stop),
		},
		{
			Name:  "info",
			Usage: "query various infomation",
			Subcommands: []cli.Command{
				{
					Name:   "cpu",
					Usage:  "display CPU info",
					Action: WithTransport(info_cpu),
				},
				{
					Name:    "memory",
					Aliases: []string{"mem"},
					Usage:   "display memory info",
					Action:  WithTransport(info_mem),
				},
				{
					Name:   "disk",
					Usage:  "display disk info",
					Action: WithTransport(info_disk),
				},
				{
					Name:   "nic",
					Usage:  "display NIC info",
					Action: WithTransport(info_nic),
				},
				{
					Name:   "os",
					Usage:  "display OS info",
					Action: WithTransport(info_os),
				},
				{
					Name:      "process",
					Aliases:   []string{"ps"},
					Usage:     "display processes info",
					ArgsUsage: "[id]",
					Action:    WithTransport(info_ps),
				},
			},
		},
		{
			Name:   "reboot",
			Usage:  "reboot the machine",
			Action: WithTransport(reboot),
		},
	}
	app.Run(os.Args)
}
