package containers

import (
	"encoding/json"
	"fmt"
	"github.com/zero-os/0-core/base/pm"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"regexp"
	"syscall"
)

const (
	backupMetaName = ".corex.meta"
)

var (
	resticSnaphostIdP = regexp.MustCompile(`snapshot ([^\s]+) saved`)
)

func (m *containerManager) backup(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Container uint16   `json:"container"`
		Repo      string   `json:"repo"`
		Password  string   `json:"password"`
		Tags      []string `json:"tags"`
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

	restic := []string{
		"-r", args.Repo,
		"backup",
		"--exclude", "proc/**",
		"--exclude", "dev/**",
	}

	for _, tag := range cont.Args.Tags {
		restic = append(restic, "--tag", tag)
	}

	for _, tag := range args.Tags {
		restic = append(restic, "--tag", tag)
	}

	root := cont.root()

	//write meta
	cargs := cont.Args
	var nics []*Nic
	for _, n := range cargs.Nics {
		if n.State == NicStateConfigured {
			nics = append(nics, n)
		}
	}
	cargs.Nics = nics
	mf := path.Join(root, backupMetaName)
	meta, err := json.Marshal(cargs)
	if err != nil {
		return nil, err
	}

	if err := ioutil.WriteFile(mf, meta, 0400); err != nil {
		return nil, err
	}

	defer os.Remove(mf)

	//we specify files to backup one by one instead of a full dire to
	//have more control
	items, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, item := range items {
		if item.Name() == "coreX" {
			continue
		}

		files = append(files, path.Join(root, item.Name()))
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("nothing to backup")
	}

	restic = append(restic, files...)

	//pause container
	syscall.Kill(-cont.PID, syscall.SIGSTOP)
	defer syscall.Kill(-cont.PID, syscall.SIGCONT)

	job, err := pm.Run(
		&pm.Command{
			Command: pm.CommandSystem,
			Arguments: pm.MustArguments(
				pm.SystemCommandArguments{
					Name:  "restic",
					Args:  restic,
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

	//read snapshot id
	match := resticSnaphostIdP.FindStringSubmatch(result.Streams.Stdout())
	if len(match) != 2 {
		return nil, fmt.Errorf("failed to retrieve snapshot ID")
	}

	return match[1], nil
}

func (c *container) restore(repo, backend string) error {
	//file://password/path/to/repo
	u, err := url.Parse(repo)
	if err != nil {
		return err
	}

	var password string
	snapshot := "latest"
	if u.Scheme == "file" {
		password = u.Host
		repo = u.Path
	} else {
		u.Fragment = ""
		repo = u.String()
	}

	snapshot = u.Fragment

	target := path.Join(backend, "ro")
	restic := []string{
		"-r", repo,
		"restore",
		"-t", target,
		snapshot,
	}

	job, err := pm.Run(
		&pm.Command{
			Command: pm.CommandSystem,
			Arguments: pm.MustArguments(
				pm.SystemCommandArguments{
					Name:  "restic",
					Args:  restic,
					StdIn: password,
				},
			),
		},
	)

	if err != nil {
		return err
	}

	if result := job.Wait(); result.State != pm.StateSuccess {
		return fmt.Errorf("failed to restore snapshot: %s", result.Streams.Stderr())
	}

	os.Remove(path.Join(target, backupMetaName))

	return nil
}
