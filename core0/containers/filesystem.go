package containers

import (
	"crypto/md5"
	"fmt"
	"github.com/g8os/core0/base/settings"
	"github.com/g8os/fs/config"
	"github.com/g8os/fs/files"
	"github.com/g8os/fs/meta"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"syscall"
)

const (
	BackendBaseDir       = "/tmp"
	ContainerBaseRootDir = "/mnt"
)

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

func (c *container) mountPList(src string, target string) error {
	//check
	hash := c.hash(src)
	backend := path.Join(BackendBaseDir, c.name(), hash)
	metaBackend := path.Join(BackendBaseDir, c.name(), fmt.Sprintf("%s+meta", hash))

	os.RemoveAll(backend)
	os.RemoveAll(metaBackend)

	for _, p := range []string{backend, metaBackend, target} {
		if err := os.MkdirAll(p, 0755); err != nil {
			return fmt.Errorf("failed to create mount points '%s': %s", p, err)
		}
	}

	plist, err := c.getPlist(src)
	if err != nil {
		return err
	}

	be := &config.Backend{Path: backend}

	ms := meta.NewMemoryMetaStore()
	//NOTE: replace the memory meta store with the file meta store for a smaller memory footprint (but slower startup)
	//ms := meta.NewFileMetaStore(metaBackend)
	if err := ms.Populate(plist, "/"); err != nil {
		return err
	}

	storageConfig := &config.StorConfig{
		URL: settings.Settings.Globals.Get("fuse_storage", "https://stor.jumpscale.org/stor2"),
	}

	storage, err := storageConfig.GetStorClient()
	if err != nil {
		return err
	}

	fs, err := files.NewFS(target, be, storage, ms, false)

	if err != nil {
		return err
	}

	go fs.Serve()

	fs.WaitMount()
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
