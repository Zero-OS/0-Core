package helper

import (
	"fmt"
	"github.com/zero-os/0-core/base/pm"
	"net/url"
)

func RestoreRepo(repo, target string, include ...string) error {
	//file://password/path/to/repo
	u, err := url.Parse(repo)
	if err != nil {
		return err
	}

	password := u.Query().Get("password")
	snapshot := u.Fragment
	if u.Scheme == "file" || len(u.Scheme) == 0 {
		repo = u.Path
	} else {
		u.Fragment = ""
		u.RawQuery = ""
		repo = u.String()
	}

	restic := []string{
		"-r", repo,
		"restore",
		"-t", target,
	}

	for _, i := range include {
		restic = append(restic, "-i", i)
	}

	restic = append(restic, snapshot)

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

	return nil
}
