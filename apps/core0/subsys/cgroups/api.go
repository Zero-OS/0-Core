package cgroups

import (
	"encoding/json"

	"github.com/zero-os/0-core/base/pm"
)

type GroupArg struct {
	Subsystem string `json:"subsystem"`
	Name      string `json:"name"`
}

func list(cmd *pm.Command) (interface{}, error) {
	return GetGroups()
}

func ensure(cmd *pm.Command) (interface{}, error) {
	var args GroupArg

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	_, err := GetGroup(args.Name, args.Subsystem)

	return nil, err
}

func remove(cmd *pm.Command) (interface{}, error) {
	var args GroupArg

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	return nil, Remove(args.Name, args.Subsystem)
}

func tasks(cmd *pm.Command) (interface{}, error) {
	var args GroupArg

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	group, err := Get(args.Name, args.Subsystem)
	if err != nil {
		return nil, pm.NotFoundError(err)
	}

	return group.Tasks()
}

func taskAdd(cmd *pm.Command) (interface{}, error) {
	var args struct {
		GroupArg
		PID int `json:"pid"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	group, err := Get(args.Name, args.Subsystem)
	if err != nil {
		return nil, pm.NotFoundError(err)
	}

	return nil, group.Task(args.PID)
}

func taskRemove(cmd *pm.Command) (interface{}, error) {
	var args struct {
		GroupArg
		PID int `json:"pid"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	group, err := Get(args.Name, args.Subsystem)
	if err != nil {
		return nil, pm.NotFoundError(err)
	}

	root := group.Root()
	return nil, root.Task(args.PID)
}
