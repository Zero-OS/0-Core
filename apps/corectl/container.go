package main

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/olekukonko/tablewriter"
	"github.com/zero-os/0-core/base/pm"
	"gopkg.in/yaml.v2"
	"os"
	"sort"
	"strconv"
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
		Content: pm.Command{
			Command: "corex.find",
			Arguments: pm.MustArguments(M{
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
	ids := make([]int, 0, len(containers))
	for id := range containers {
		iid, _ := strconv.ParseInt(id, 10, 32)
		ids = append(ids, int(iid))
	}
	sort.Ints(ids)

	for _, id := range ids {
		sid := fmt.Sprintf("%d", id)
		container := containers[sid]
		table.Append([]string{
			sid,
			container.Container.Arguments.Root,
			container.Container.Arguments.Hostname,
			strings.Join(container.Container.Arguments.Tags, ", "),
		})
	}

	table.Render()
}

func containerInspect(t Transport, c *cli.Context) {
	id := c.Args().First()
	if id == "" {
		log.Fatal("missing container id")
	}

	response, err := t.Run(Command{
		Sync: true,
		Content: pm.Command{
			Command:   "corex.list",
			Arguments: pm.MustArguments(M{}),
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	response.ValidateResultOrExit()
	var containers map[string]interface{}
	if err := json.Unmarshal([]byte(response.Data), &containers); err != nil {
		log.Fatal(err)
	}

	container, ok := containers[id]
	if !ok {
		log.Fatalf("no container with id: %s", id)
	}

	data, _ := yaml.Marshal(container)
	fmt.Println(string(data))
}
