package main

import (
	"github.com/codegangsta/cli"
	"github.com/g8os/core0/base/pm/core"
)

func jobs(t Transport, c *cli.Context) {
	response, err := t.Run(Command{
		Sync: true,
		Content: core.Command{
			Command:   "job.list",
			Arguments: core.MustArguments(M{}),
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	response.ValidateResultOrExit()
	response.PrintYaml()
}

func jobKill(t Transport, c *cli.Context) {
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
