package main

import (
	"fmt"
	"github.com/g8os/core0/base/pm/core"
	"gopkg.in/yaml.v2"
	"io"
	"os"
)

var (
	outputs = []io.Writer{
		os.Stdout,
		os.Stderr,
	}
)

type M map[string]interface{}

type Response core.JobResult

func (r *Response) Print() {
	data, err := yaml.Marshal(r)
	if err != nil {
		log.Fatalf("failed to format results: %s", err)
	}

	fmt.Println(string(data))
}

func (r *Response) PrintStreams() {
	for i, s := range r.Streams {
		if len(s) > 0 {
			fmt.Fprintf(outputs[i], s)
		}
	}
}

func (r *Response) ValidateResultOrExit() {

	if r.State != core.StateSuccess {
		log.Errorf("State: %s", r.State)
		if r.Data != "" {
			log.Errorf("%s", r.Data)
		}

		if r.Critical != "" {
			log.Errorf("%s", r.Critical)
		}

		os.Exit(1)
	}
}
