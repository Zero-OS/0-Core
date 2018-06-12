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

}
