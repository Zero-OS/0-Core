package containers

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/g8os/core0/base/settings"
	"github.com/g8os/g8ufs"
	"github.com/g8os/g8ufs/meta"
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

//a helper to close all under laying readers in a plist file stream since decompression doesn't
//auto close the under laying layer.
type underLayingCloser struct {
	readers []io.Reader
}

//close all layers.
func (u *underLayingCloser) Close() error {
	for i := len(u.readers) - 1; i >= 0; i-- {
		r := u.readers[i]
		if c, ok := r.(io.Closer); ok {
			c.Close()
		}
	}

	return nil
}

//read only from the last layer.
func (u *underLayingCloser) Read(p []byte) (int, error) {
	return u.readers[len(u.readers)-1].Read(p)
}

func (c *container) getMetaDBTar(src string) (io.ReadCloser, error) {
	u, err := url.Parse(src)
	if err != nil {
		return nil, err
	}

	var reader io.ReadCloser
	base := path.Base(u.Path)

	if u.Scheme == "file" || u.Scheme == "" {
		// check file exists
		_, err := os.Stat(u.Path)
		if err != nil {
			return nil, err
		}
		reader, err = os.Open(u.Path)
		if err != nil {
			return nil, err
		}
	} else if u.Scheme == "http" || u.Scheme == "https" {
		response, err := http.Get(src)
		if err != nil {
			return nil, err
		}

		reader = response.Body
	} else {
		return nil, fmt.Errorf("invalid plist url (%s)", src)
	}

	var closer underLayingCloser
	closer.readers = append(closer.readers, reader)

	ext := path.Ext(base)
	switch ext {
	case ".tgz":
		fallthrough
	case ".gz":
		if r, err := gzip.NewReader(reader); err != nil {
			closer.Close()
			return nil, err
		} else {
			closer.readers = append(closer.readers, r)
		}
		return &closer, nil
	case ".tbz2":
		fallthrough
	case ".bz2":
		closer.readers = append(closer.readers, bzip2.NewReader(reader))
		return &closer, err
	case ".tar":
		return &closer, nil
	}

	return nil, fmt.Errorf("unknown plist format %s", ext)
}

func (c *container) getMetaDB(src string) (string, error) {
	reader, err := c.getMetaDBTar(src)
	if err != nil {
		return "", err
	}

	defer reader.Close()

	archive := tar.NewReader(reader)
	db := path.Join(BackendBaseDir, c.name(), fmt.Sprintf("%s.db", c.hash(src)))
	log.Debugf("Extracting meta to %s", db)
	if err := os.MkdirAll(db, 0755); err != nil {
		return "", err
	}

	for {
		header, err := archive.Next()
		if err != nil && err != io.EOF {
			return "", err
		} else if err == io.EOF {
			break
		}

		if header.FileInfo().IsDir() {
			continue
		}

		base := path.Join(db, path.Dir(header.Name))
		log.Debugf("extracting: %s", header.Name)
		if err := os.MkdirAll(base, 0755); err != nil {
			return "", err
		}

		file, err := os.Create(path.Join(db, header.Name))
		if err != nil {
			return "", err
		}

		if _, err := io.Copy(file, archive); err != nil {
			file.Close()
			return "", err
		}

		file.Close()
	}

	return db, nil
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

	db, err := c.getMetaDB(src)
	if err != nil {
		return err
	}

	store, err := meta.NewRocksMeta("", db)
	if err != nil {
		return err
	}

	u, err := url.Parse(settings.Settings.Globals.Get("fuse_storage", "ardb://home.maxux.net:26379"))
	if err != nil {
		return err
	}

	storage, err := storage.NewARDBStorage(u)
	if err != nil {
		return err
	}

	fs, err := g8ufs.Mount(&g8ufs.Options{
		Backend:   backend,
		Target:    target,
		Storage:   storage,
		MetaStore: store,
		Reset:     true,
		Exec:      c.exec,
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
