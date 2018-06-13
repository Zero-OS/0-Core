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

//actions to control group params
func cpusetReset(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	group, err := Get(args.Name, "cpuset")

	if err != nil {
		return nil, pm.NotFoundError(err)
	}

	if group, ok := group.(CPUSetGroup); ok {
		group.Reset()
		return nil, nil
	}

	return nil, pm.InternalError(ErrInvalidType)
}

func cpusetSpec(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Name string `json:"name,omitempty"`
		Cpus string `json:"cpus"`
		Mems string `json:"mems"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	group, err := Get(args.Name, "cpuset")

	if err != nil {
		return nil, pm.NotFoundError(err)
	}

	if group, ok := group.(CPUSetGroup); ok {
		if len(args.Cpus) != 0 {
			if err := group.Cpus(args.Cpus); err != nil {
				return nil, err
			}
		}

		if len(args.Mems) != 0 {
			if err := group.Mems(args.Mems); err != nil {
				return nil, err
			}
		}

		args.Name = ""
		args.Cpus, _ = group.GetCpus()
		args.Mems, _ = group.GetMems()

		return args, nil
	}

	return nil, pm.InternalError(ErrInvalidType)
}
