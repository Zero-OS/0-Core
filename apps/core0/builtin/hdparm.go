package builtin

import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/zero-os/0-core/base/pm"
)

func exists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil {
		 return true, nil
	 }
    if os.IsNotExist(err) {
		 return false, nil 
	}
    return true, nil 
}

func init() {
	pm.RegisterBuiltIn("core.spindown_disk", hdparmSpindownDisk)
}

func hdparmSpindownDisk(cmd *pm.Command) (interface{}, error) {
	var args struct {
		DiskPath string `json:"disk_path"`
		Spindown int	`json:"spindown"`
	}
	cmdbin := "hdparm"
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	// assert disk exists
	_, err := exists(args.DiskPath)
	if err!= nil {
		return nil, pm.BadRequestError(fmt.Errorf("Disk doesn't exist: %s", args.DiskPath))
	
	}
	if !(args.Spindown>0 && args.Spindown<241){
		return nil, pm.BadRequestError(fmt.Errorf("Spindown %d out of range 0 - 240", args.Spindown))
	
	} 

	_, err = pm.System(cmdbin, "-S", fmt.Sprintf("%d", args.Spindown), args.DiskPath)

	if err != nil {
		return nil, err
	}
	return nil, nil
}
