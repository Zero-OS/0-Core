package builtin

import (
	"encoding/json"
	"fmt"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/process"
	"github.com/zero-os/0-core/base/pm/stream"
)

func init() {
	pm.CmdMap["core.subscribe"] = process.NewInternalProcessFactoryWithCtx(subscribe)
}

func subscribe(ctx *process.Context) (interface{}, error) {
	var args struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(*ctx.Command.Arguments, &args); err != nil {
		return nil, err
	}

	runner, ok := pm.GetManager().Runner(args.ID)

	if !ok {
		return nil, fmt.Errorf("job '%s' does not exist", args.ID)
	}

	runner.Subscribe(func(msg *stream.Message) {
		ctx.Message(msg)
	})

	runner.Wait()
	return nil, nil
}
