package main

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/g8os/core0/base/pm/core"
	"gopkg.in/yaml.v2"
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
		Content: core.Command{
			Command:   cmd,
			Arguments: core.MustArguments(data),
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	response.ValidateResultOrExit()

	var output interface{}
	if err := json.Unmarshal([]byte(response.Data), &output); err != nil {
		log.Fatal(err)
	}

	if out, err := yaml.Marshal(output); err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(string(out))
	}
}

func info_cpu(t Transport, c *cli.Context) {
	info(t, "get_cpu_info")
}

func info_disk(t Transport, c *cli.Context) {
	info(t, "get_disk_info")
}

func info_mem(t Transport, c *cli.Context) {
	info(t, "get_mem_info")
}

func info_nic(t Transport, c *cli.Context) {
	info(t, "get_nic_info")
}

func info_os(t Transport, c *cli.Context) {
	info(t, "get_os_info")
}

func info_ps(t Transport, c *cli.Context) {
	id := c.Args().First()

	if id != "" {
		info(t, "get_process_stats", M{
			"id": id,
		})
	} else {
		info(t, "get_processes_stats", M{})
	}

}
