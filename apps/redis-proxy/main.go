package main

import (
	"os"

	"github.com/codegangsta/cli"
	logging "github.com/op/go-logging"
	"github.com/zero-os/0-core/base/utils"
)

var (
	log = logging.MustGetLogger("redis-proxy")
)

func main() {
	app := cli.NewApp()

	app.Name = "redis-proxy"
	app.Usage = "add tls and custom authentication on top of redis"
	app.Version = "1.0"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "organization, o",
			Value: "",
			Usage: "IYO organization that has to be valid in the jwt calims, if not provided, it will be parsed from kerenel cmdline, otherwise no authentication will be applied",
		},
		cli.StringFlag{
			Name:  "listen, l",
			Value: "0.0.0.0:6379",
			Usage: "listing address (default: 0.0.0.0:6379)",
		},
		cli.StringFlag{
			Name:  "redis, r",
			Value: "/var/run/redis.sock",
			Usage: "redis unix socket to proxy",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug logging",
		},
	}

	app.Before = func(ctx *cli.Context) error {
		if ctx.GlobalBool("debug") {
			logging.SetLevel(logging.DEBUG, "")
		} else {
			logging.SetLevel(logging.INFO, "")
		}

		return nil
	}

	app.Action = func(ctx *cli.Context) error {
		organization := ctx.String("organization")
		if organization == "" {
			if orgs, ok := utils.GetKernelOptions().Get("organization"); ok {
				organization = orgs[len(orgs)-1]
			}
		}

		return Proxy(ctx.String("listen"), ctx.String("redis"), organization)
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
