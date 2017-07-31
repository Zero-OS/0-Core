package main

import (
	"github.com/codegangsta/cli"
	"github.com/zero-os/0-core/base/pm"
)

func info(t Transport, cmd string, body ...interface{}) {
	var data interface{}
	switch len(body) {
	case 0:
	case 1:
		data = body[0]
	default:
		panic("info can only take one optional data argument")
	}

	response, err := t.Run(Command{
		Sync: true,
		Content: pm.Command{
			Command:   cmd,
			Arguments: pm.MustArguments(data),
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	response.ValidateResultOrExit()
	response.PrintYaml()
}

func info_cpu(t Transport, c *cli.Context) {
	info(t, "info.cpu")
}

func info_disk(t Transport, c *cli.Context) {
	info(t, "info.disk")
}

func info_mem(t Transport, c *cli.Context) {
	info(t, "info.mem")
}

func info_nic(t Transport, c *cli.Context) {
	info(t, "info.nic")
}

func info_os(t Transport, c *cli.Context) {
	info(t, "info.os")
}
