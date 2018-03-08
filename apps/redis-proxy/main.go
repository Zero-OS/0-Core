package main

import (
	"os"

	"github.com/codegangsta/cli"
	logging "github.com/op/go-logging"
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
			Usage: "IYO organization that has to be valid in the jwt calims, no authenticaion is required if not set",
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
	}

	app.Action = func(ctx *cli.Context) error {
		return Proxy(ctx.String("listen"), ctx.String("redis"), ctx.String("organization"))
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
