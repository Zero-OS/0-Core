package plugin

import (
	"encoding/json"
	"fmt"
	"os"
)

type Runnable func(args json.RawMessage) (interface{}, error)

type Commands map[string]Runnable

func output(o interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	os.Stdout.WriteString("20:::")
	encoder.Encode(o)
	os.Stdout.WriteString(":::")
}

func exitError(e error) {
	output(e.Error())
	os.Exit(1)
}

// Plugin is the only thing u need to call from your plugin with the supported commands.
func Plugin(commands Commands) {
	defer func() {
		if err := recover(); err != nil {
			exitError(fmt.Errorf("paniced with error: %v", err))
		}
	}()

	args := os.Args[1:]

	if len(args) == 0 {
		exitError(fmt.Errorf("missing command"))
	}

	cmd := args[0]

	var input json.RawMessage
	if len(args) == 1 {
		in := args[0]
		input = json.RawMessage([]byte(in))
	}

	if handler, ok := commands[cmd]; ok {
		out, err := handler(input)
		if err != nil {
			exitError(err)
		}
		output(out)
	} else {
		exitError(fmt.Errorf("unknown command: %s", cmd))
	}

	os.Exit(0)
}
