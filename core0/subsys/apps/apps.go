package apps

import (
	"encoding/json"
	"fmt"
	"github.com/pborman/uuid"
	"github.com/zero-os/0-core/base/g8ufs"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"github.com/zero-os/0-core/core0/transport"
	"os"
	"path"
	"strings"
	"syscall"
)

const (
	AppsBaseRootDir = "/opt"
	BackendBaseDir  = "/var/cache/apps"
)

type appMgr struct {
	sink *transport.Sink
}

func AppsSubsystem(sink *transport.Sink) error {
	mgr := &appMgr{
		sink: sink,
	}

	pm.CmdMap["core.app"] = process.NewInternalProcessFactory(mgr.app)

	return nil
}

func (a *appMgr) app(cmd *core.Command) (interface{}, error) {
	var args struct {
		FList   string                         `json:"flist"`
		Storage string                         `json:"storage"`
		ID      string                         `json:"id"`
		Command process.SystemCommandArguments `json:"command"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	id := fmt.Sprintf("app-%s", cmd.ID)
	target := path.Join(AppsBaseRootDir, id)
	backend := path.Join(BackendBaseDir, id)
	cache := path.Join(BackendBaseDir, "cache")

	if err := g8ufs.Mount(args.FList, target, backend, args.Storage, cache); err != nil {
		return nil, err
	}

	//start the actual command
	job := args.ID
	if len(job) == 0 {
		job = uuid.New()
	}

	//rewrite path to binary
	args.Command.Name = path.Join(target, strings.TrimRight(args.Command.Name, "/"))
	a.sink.Flag(job)
	_, err := pm.GetManager().RunCmd(
		&core.Command{
			ID:        job,
			Command:   process.CommandSystem,
			Arguments: core.MustArguments(args.Command),
			Tags:      cmd.Tags,
		},
		&pm.ExitHook{
			Action: func(_ bool) {
				syscall.Unmount(target, syscall.MNT_DETACH|syscall.MNT_FORCE)
				os.RemoveAll(backend)
				os.RemoveAll(target)
			},
		},
	)

	if err != nil {
		return nil, err
	}

	return job, nil
}
