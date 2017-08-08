package containers

import (
	"encoding/json"
	"fmt"
	"github.com/zero-os/0-core/base/pm"
	"syscall"
)

func (m *containerManager) backup(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Container uint16 `json:"container"`
		Repo      string `json:"repo"`
		Password  string `json:"password"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	if args.Container <= 0 {
		return nil, fmt.Errorf("invalid container id")
	}

	m.conM.RLock()
	cont, ok := m.containers[args.Container]
	m.conM.RUnlock()

	if !ok {
		return nil, fmt.Errorf("container does not exist")
	}

	//pause container
	//TODO: avoid race if cont has just started and pid is not set yet!
	if cont.PID == 0 {
		return nil, fmt.Errorf("container is not fully started yet")
	}

	//pause container
	syscall.Kill(-cont.PID, syscall.SIGSTOP)
	defer syscall.Kill(-cont.PID, syscall.SIGCONT)

	job, err := pm.Run(
		&pm.Command{
			Command: pm.CommandSystem,
			Arguments: pm.MustArguments(
				pm.SystemCommandArguments{
					Name: "restic",
					Args: []string{
						"-r", args.Repo,
						"backup",
						"--one-file-system", //do not backup other mounts under this?
						"--exclude", "proc/**",
						"--exclude", "dev/**",
						cont.root(),
					},
					StdIn: args.Password,
				},
			),
		},
	)

	if err != nil {
		return nil, err
	}

	result := job.Wait()
	if result.State != pm.StateSuccess {
		return nil, fmt.Errorf("failed to backup container: %s", result.Streams.Stderr())
	}

	return nil, nil
}
