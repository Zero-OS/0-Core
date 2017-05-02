package main

import (
	"encoding/json"
	"github.com/codegangsta/cli"
	"github.com/g8os/core0/base/pm/core"
	"github.com/olekukonko/tablewriter"
	"os"
	"strings"
)

type containerData struct {
	Container struct {
		Arguments struct {
			Root     string   `json:"root"`
			Hostname string   `json:"hostname"`
			Tags     []string `json:"tags"`
		} `json:"arguments"`
		PID  int    `json:"pid"`
		Root string `json:"root"`
	} `json:"container"`
}

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
	var containers map[string]containerData
	if err := json.Unmarshal([]byte(response.Data), &containers); err != nil {
		log.Fatal(err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorders(tablewriter.Border{})
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader([]string{"ID", "FLIST", "HOSTNAME", "TAGS"})
	for id, container := range containers {
		table.Append([]string{
			id,
			container.Container.Arguments.Root,
			container.Container.Arguments.Hostname,
			strings.Join(container.Container.Arguments.Tags, ", "),
		})
	}

	table.Render()
}
