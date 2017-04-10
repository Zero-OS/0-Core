package main

import (
	"github.com/codegangsta/cli"
	"github.com/g8os/core0/base/pm/core"
)

func ping(t Transport, c *cli.Context) {
	response, err := t.Run(Command{
		Sync: true,
		Content: core.Command{
			Command: "core.ping",
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	response.Print()
}
