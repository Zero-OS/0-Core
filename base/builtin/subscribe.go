package builtin

import (
	"encoding/json"
	"fmt"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/stream"
)

func init() {
	pm.RegisterBuiltInWithCtx("core.subscribe", subscribe)
}

func subscribe(ctx *pm.Context) (interface{}, error) {
	var args struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(*ctx.Command.Arguments, &args); err != nil {
		return nil, err
	}

	job, ok := pm.JobOf(args.ID)

	if !ok {
		return nil, fmt.Errorf("job '%s' does not exist", args.ID)
	}

	job.Subscribe(func(msg *stream.Message) {
		ctx.Message(msg)
	})

	job.Wait()
	return nil, nil
}
