package builtin

import (
	"encoding/json"
	"fmt"

	"github.com/zero-os/0-core/base/pm"
)

const (
	cmdFlistCreate = "flist.create"
)

type flistMgr struct{}

func init() {
	flist := flistMgr{}
	pm.RegisterBuiltIn(cmdFilesystemOpen, flist.create)
}

type createArgs struct {
	Flist   string //path where to create the flist
	Storage string //host:port to the data storage
	Src     string //path to the directory to create flist from
}

func (c createArgs) Validate() error {
	if c.Flist == "" {
		return fmt.Errorf("flist destination need to be specified")
	}
	if c.Storage == "" {
		return fmt.Errorf("flist data storage need to be specified")
	}
	if c.Src == "" {
		return fmt.Errorf("source directory need to be specified")
	}
	return nil
}

func zflist(args ...string) (*pm.JobResult, error) {
	log.Debugf("zflist %v", args)
	return pm.System("zflist", args...)
}

func (f *flistMgr) create(cmd *pm.Command) (interface{}, error) {
	var args createArgs

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	if err := args.Validate(); err != nil {
		return nil, err
	}

	_, err := zflist("--archive", args.Flist, "--create", args.Src, "--backend", args.Storage)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
