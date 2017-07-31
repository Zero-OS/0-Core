package main

import (
	"github.com/codegangsta/cli"
	"github.com/zero-os/0-core/base/pm/core"
)

func reboot(t Transport, c *cli.Context) {
	response, err := t.Run(Command{
		Sync: true,
		Content: pm.Command{
			Command: "core.reboot",
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	//you probably won't reach here but let's assume you did
	response.ValidateResultOrExit()
}

func poweroff(t Transport, c *cli.Context) {
	response, err := t.Run(Command{
		Sync: true,
		Content: pm.Command{
			Command: "core.poweroff",
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	//you probably won't reach here but let's assume you did
	response.ValidateResultOrExit()
}
