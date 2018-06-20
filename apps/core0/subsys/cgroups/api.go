package cgroups

import (
	"encoding/json"

	"github.com/zero-os/0-core/base/pm"
)

type GroupArg struct {
	Subsystem Subsystem `json:"subsystem"`
	Name      string    `json:"name"`
}

func list(cmd *pm.Command) (interface{}, error) {
	return GetGroups()
}

func ensure(cmd *pm.Command) (interface{}, error) {
	var args GroupArg

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	_, err := GetGroup(args.Subsystem, args.Name)

	return nil, err
}

func remove(cmd *pm.Command) (interface{}, error) {
	var args GroupArg

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	return nil, Remove(args.Subsystem, args.Name)
}

func reset(cmd *pm.Command) (interface{}, error) {
	var args GroupArg

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	group, err := Get(args.Subsystem, args.Name)
	if err != nil {
		return nil, pm.NotFoundError(err)
	}

	group.Reset()

	return nil, nil
}

func tasks(cmd *pm.Command) (interface{}, error) {
	var args GroupArg

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	group, err := Get(args.Subsystem, args.Name)
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

	group, err := Get(args.Subsystem, args.Name)
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

	group, err := Get(args.Subsystem, args.Name)
	if err != nil {
		return nil, pm.NotFoundError(err)
	}

	root := group.Root()
	return nil, root.Task(args.PID)
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

	group, err := Get(CPUSetSubsystem, args.Name)

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

func memorySpec(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Name string `json:"name,omitempty"`
		Mem  int    `json:"mem"`
		Sawp int    `json:"swap"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, pm.BadRequestError(err)
	}

	group, err := Get(MemorySubsystem, args.Name)

	if err != nil {
		return nil, pm.NotFoundError(err)
	}

	if group, ok := group.(MemoryGroup); ok {
		if args.Mem != 0 {
			if err := group.Limit(args.Mem, args.Sawp); err != nil {
				return nil, err
			}
		}

		args.Name = ""
		mem, swap, _ := group.Limits()
		args.Mem = mem
		args.Sawp = swap - mem

		return args, nil
	}

	return nil, pm.InternalError(ErrInvalidType)
}
