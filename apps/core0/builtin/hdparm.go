package builtin

import (
	"encoding/json"
	"fmt"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/utils"
)

func init() {
	pm.RegisterBuiltIn("core.spindown_disk", hdparmSpindownDisk)
}

func hdparmSpindownDisk(cmd *pm.Command) (interface{}, error) {
	var args struct {
		DiskPath string `json:"disk_path"`
		Spindown uint	`json:"spindown"`
	}
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	// assert disk exists
	if !utils.Exists(args.DiskPath){
		return nil, pm.BadRequestError(fmt.Errorf("disk doesn't exist: %s", args.DiskPath))

	}
	if !(args.Spindown<241){
		return nil, pm.BadRequestError(fmt.Errorf("spindown %d out of range 1 - 240", args.Spindown))
	
	} 
	_, err := pm.System("hdparm", "-S", fmt.Sprintf("%d", args.Spindown), args.DiskPath)

	if err != nil {
		return nil, err
	}

	return nil, nil
}
