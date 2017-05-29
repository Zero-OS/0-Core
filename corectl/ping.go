package main

import (
	"github.com/codegangsta/cli"
	"github.com/zero-os/0-core/base/pm/core"
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
