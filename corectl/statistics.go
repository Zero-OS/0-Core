package main

import (
	"strings"

	"fmt"
	"github.com/codegangsta/cli"
	"github.com/zero-os/0-core/base/pm"
)

func statistics(t Transport, c *cli.Context) {
	tags := make(map[string]string)
	var key string

	if c.Args().Present() {
		key = c.Args().First()
		for _, tag := range c.Args().Tail() {
			splits := strings.Split(tag, "=")
			if len(splits) != 2 {
				log.Fatalf("Tag %v has an incorrect format", tag)
			}
			tags[splits[0]] = splits[1]
		}
	}

	response, err := t.Run(Command{
		Sync: true,
		Content: pm.Command{
			Command: "aggregator.query",
			Arguments: pm.MustArguments(M{
				"key":  key,
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
