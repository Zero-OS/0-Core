package containers

import (
	"crypto/md5"
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/g8os/core0/base/settings"
	"github.com/g8os/g8ufs"
	"github.com/g8os/g8ufs/storage"
	"github.com/pborman/uuid"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"syscall"
)

const (
	BackendBaseDir       = "/tmp"
	ContainerBaseRootDir = "/mnt"
)

type starterWrapper struct {
	cmd *core.Command
	run pm.Runner
}

func (s *starterWrapper) Start() error {
	runner, err := pm.GetManager().RunCmd(s.cmd)
	s.run = runner
	return err
}

func (s *starterWrapper) Wait() error {
	if s.run == nil {
		return fmt.Errorf("not started")
	}
	r := s.run.Wait()
	if r.State != core.StateSuccess {
		return fmt.Errorf("exit error: %s", r.Streams[1])
	}
	return nil
}

func (c *container) name() string {
	return fmt.Sprintf("container-%d", c.id)
}

func (c *container) getPlist(src string) (string, error) {
	u, err := url.Parse(src)
	if err != nil {
		return "", err
	}

	if u.Scheme == "file" || u.Scheme == "" {
		// check file exists
		_, err := os.Stat(u.Path)
		if err != nil {
			return "", err
		}
		return u.Path, nil
	} else if u.Scheme == "http" || u.Scheme == "https" {
		response, err := http.Get(src)
		if err != nil {
			return "", err
		}

		defer response.Body.Close()

		base := path.Base(u.Path)
		name := path.Join(BackendBaseDir, c.name(), fmt.Sprintf("%s.plist", base))

		file, err := os.Create(name)
		if err != nil {
			return "", err
		}
		defer file.Close()
		if _, err := io.Copy(file, response.Body); err != nil {
			return "", nil
		}

		return name, nil
	}

	return "", fmt.Errorf("invalid plist url %s", src)
}

func (c *container) exec(name string, arg ...string) g8ufs.Starter {
	return &starterWrapper{
		cmd: &core.Command{
			ID:      uuid.New(),
			Command: process.CommandSystem,
			Arguments: core.MustArguments(
				process.SystemCommandArguments{
					Name: name,
					Args: arg,
				},
			),
		},
	}
}

func (c *container) mountPList(src string, target string) error {
	//check
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	hash := c.hash(src)
	backend := path.Join(BackendBaseDir, c.name(), hash)

	os.RemoveAll(backend)
	os.MkdirAll(backend, 0755)

	plist, err := c.getPlist(src)
	if err != nil {
		return err
	}

	u, err := url.Parse(settings.Settings.Globals.Get("fuse_storage", "https://stor.jumpscale.org/stor2"))
	if err != nil {
		return err
	}

	aydo, err := storage.NewAydoStorage(u)
	if err != nil {
		return err
	}

	fs, err := g8ufs.Mount(&g8ufs.Options{
		Backend: backend,
		PList:   plist,
		Target:  target,
		Storage: aydo,
		Reset:   true,
		Exec:    c.exec,
	})

	if err != nil {
		return err
	}

	go func() {
		err := fs.Wait()
		if err != nil {
			switch e := err.(type) {
			case *exec.ExitError:
				log.Errorf("unionfs exited with err: %s", e)
				log.Debugf("%s", string(e.Stderr))
			default:
				log.Errorf("unionfs exited with err: %s", e)
			}
		}
	}()

	return nil
}

func (c *container) hash(src string) string {
	m := md5.New()
	io.WriteString(m, src)
	return fmt.Sprintf("%x", m.Sum(nil))
}

func (c *container) root() string {
	return path.Join(ContainerBaseRootDir, c.name())
}

func (c *container) mount() error {
	//mount root plist.
	//prepare root folder.
	root := c.root()
	log.Debugf("Container root: %s", root)
	os.RemoveAll(root)

	if err := c.mountPList(c.args.Root, root); err != nil {
		return err
	}

	for src, dst := range c.args.Mount {
		target := path.Join(root, dst)
		if err := os.MkdirAll(target, 0755); err != nil {
			return err
		}
		//src can either be a location on HD, or another plist
		u, err := url.Parse(src)
		if err != nil {
			log.Errorf("bad mount source '%s'", u)
		}

		if u.Scheme == "" {
			if err := syscall.Mount(src, target, "", syscall.MS_BIND, ""); err != nil {
				return err
			}
		} else {
			//assume a plist
			if err := c.mountPList(src, target); err != nil {
				return err
			}
		}
	}

	return nil
}
