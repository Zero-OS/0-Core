package main

import (
	"github.com/codegangsta/cli"
	"github.com/zero-os/0-core/base/pm/core"
)

func system(t Transport, c *cli.Context) {
	if c.Args().First() == "" {
		log.Fatalf("missing command to execute")
		return
	}
	sync := !c.GlobalBool("async")
	response, err := t.Run(Command{
		Sync:      sync,
		Container: c.GlobalString("container"),
		Content: core.Command{
			Command: "core.system",
			Arguments: core.MustArguments(M{
				"name": c.Args().First(),
				"args": c.Args().Tail(),
			}),
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	if sync {
		response.PrintStreams()
		response.ValidateResultOrExit()
	} else {
		log.Infof("Job started with ID: %s", response.ID)
	}
}
