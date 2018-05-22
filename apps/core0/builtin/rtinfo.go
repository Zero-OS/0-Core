package builtin

import (
	"encoding/json"
	"fmt"

	"github.com/zero-os/0-core/base/pm"
)

func init() {
	pm.RegisterBuiltIn("core.rtinfo", rtinfoRun)
}

func rtinfoRun(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Host  string   `json:"host`
		Port  uint     `json:"port"`
		Disks []string `json:"disks"`
	}
	cmdbin := "rtinfo-client"
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	var cmdargs []string
	cmdargs = append(cmdargs, "--host", args.Host, "--port", fmt.Sprintf("%d", args.Port))

	for _, d := range args.Disks {
		cmdargs = append(cmdargs, "--disk", d)
	}

	if _, err := pm.System(cmdbin, cmdargs...); err != nil {
		return nil, err
	}

	return nil, nil

}
