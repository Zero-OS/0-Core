package main

import (
	"github.com/codegangsta/cli"
	"github.com/g8os/core0/base/pm/core"
)

func stop(t Transport, c *cli.Context) {
	id := c.Args().First()
	if id == "" {
		log.Fatal("wrong usage")
	}

	response, err := t.Run(Command{
		Sync: true,
		Content: core.Command{
			Command: "job.kill",
			Arguments: core.MustArguments(M{
				"id": id,
			}),
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	response.ValidateResultOrExit()
}
