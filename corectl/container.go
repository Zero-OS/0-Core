package main

import (
	"github.com/codegangsta/cli"
	"github.com/g8os/core0/base/pm/core"
)

func containers(t Transport, c *cli.Context) {
	var tags []string
	if c.Args().Present() {
		tags = append(tags, c.Args().First())
		tags = append(tags, c.Args().Tail()...)
	}

	response, err := t.Run(Command{
		Sync: true,
		Content: core.Command{
			Command: "corex.find",
			Arguments: core.MustArguments(M{
				"tags": tags,
			}),
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	response.ValidateResultOrExit()
	response.PrintYaml()
}
