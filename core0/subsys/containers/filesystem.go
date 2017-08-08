package containers

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"github.com/pborman/uuid"
	"github.com/shirou/gopsutil/disk"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/stream"
	"github.com/zero-os/0-core/base/settings"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
)

const (
	BackendBaseDir       = "/var/cache/containers"
	ContainerBaseRootDir = "/mnt"
)

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

		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to download flist: %s", response.Status)
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
	case ".flist":
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

func (c *container) mountPList(src string, target string, hooks ...pm.RunnerHook) error {
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

	storageUrl := c.Args.Storage
	if storageUrl == "" {
		storageUrl = settings.Settings.Globals.Get("storage", "ardb://hub.gig.tech:16379")
		c.Args.Storage = storageUrl
	}

	cache := settings.Settings.Globals.Get("cache", path.Join(BackendBaseDir, "cache"))
	cmd := &pm.Command{
		ID:       uuid.New(),
		Command:  pm.CommandSystem,
		NoOutput: false, //this can't be set to true other wise the MatchHook below won't work
		Arguments: pm.MustArguments(pm.SystemCommandArguments{
			Name: "g8ufs",
			Args: []string{
				"-reset",
				"-backend", backend,
				"-cache", cache,
				"-meta", db,
				"-storage-url", storageUrl,
				target},
		}),
	}

	var o sync.Once
	var wg sync.WaitGroup
	wg.Add(1)

	hooks = append(hooks, &pm.MatchHook{
		Match: "mount starts",
		Action: func(_ *stream.Message) {
			o.Do(wg.Done)
		},
	}, &pm.ExitHook{
		Action: func(s bool) {
			log.Debugf("mount point '%s' exited with '%v'", target, s)
			o.Do(func() {
				if !s {
					err = fmt.Errorf("upnormal exit of filesystem mount at '%s'", target)
				}
				wg.Done()
			})
		},
	})

	pm.Run(cmd, hooks...)

	//wait for either of the hooks (ready or exit)
	wg.Wait()
	return err
}

func (c *container) hash(src string) string {
	m := md5.New()
	io.WriteString(m, src)
	return fmt.Sprintf("%x", m.Sum(nil))
}

func (c *container) root() string {
	return path.Join(ContainerBaseRootDir, c.name())
}

type SortableDisks []disk.PartitionStat

func (d SortableDisks) Len() int {
	return len(d)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (d SortableDisks) Less(i, j int) bool {
	return len(d[i].Mountpoint) > len(d[j].Mountpoint)
}

// Swap swaps the elements with indexes i and j.
func (d SortableDisks) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (c *container) getFSType(dir string) string {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}

	dir = strings.TrimRight(dir, "/") + "/"

	parts, err := disk.Partitions(true)
	if err != nil {
		return ""
	}

	sort.Sort(SortableDisks(parts))

	for _, part := range parts {
		mountpoint := part.Mountpoint
		if mountpoint != "/" {
			mountpoint += "/"
		}

		if strings.Index(dir, mountpoint) == 0 {
			return part.Fstype
		}
	}

	return ""
}

func (c *container) sandbox() error {
	//mount root plist.
	//prepare root folder.

	//make sure we remove the directory
	os.RemoveAll(path.Join(BackendBaseDir, c.name()))
	fstype := c.getFSType(BackendBaseDir)
	log.Debugf("Sandbox fileystem type: %s", fstype)

	if fstype == "btrfs" {
		//make sure we delete it if sub volume exists
		pm.System("btrfs", "subvolume", "delete", path.Join(BackendBaseDir, c.name()))
		pm.System("btrfs", "subvolume", "create", path.Join(BackendBaseDir, c.name()))
	}

	root := c.root()
	log.Debugf("Container root: %s", root)
	os.RemoveAll(root)

	onSBExit := &pm.ExitHook{
		Action: func(_ bool) {
			c.cleanSandbox()
		},
	}

	if err := c.mountPList(c.Args.Root, root, onSBExit); err != nil {
		return fmt.Errorf("mount-root-plist(%s)", err)
	}

	for src, dst := range c.Args.Mount {
		target := path.Join(root, dst)
		if err := os.MkdirAll(target, 0755); err != nil {
			return fmt.Errorf("mkdirAll(%s)", err)
		}
		//src can either be a location on HD, or another plist
		u, err := url.Parse(src)
		if err != nil {
			return fmt.Errorf("bad mount source '%s': %s", src, err)
		}

		if u.Scheme == "" {
			if err := syscall.Mount(src, target, "", syscall.MS_BIND, ""); err != nil {
				return fmt.Errorf("mount-bind(%s)", err)
			}
		} else {
			//assume a plist
			if err := c.mountPList(src, target); err != nil {
				return fmt.Errorf("mount-bind-plist(%s)", err)
			}
		}
	}

	coreXTarget := path.Join(root, coreXBinaryName)
	if f, err := os.Create(coreXTarget); err == nil {
		f.Close()
	} else {
		log.Errorf("Failed to touch file '%s': %s", coreXTarget, err)
	}

	coreXSrc, err := exec.LookPath(coreXBinaryName)
	if err != nil {
		return err
	}

	if err := syscall.Mount(coreXSrc, coreXTarget, "", syscall.MS_BIND, ""); err != nil {
		return err
	}

	return nil
}

func (c *container) unMountAll() error {
	mnts, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return err
	}
	root := c.root()
	var targets []string
	for _, line := range strings.Split(string(mnts), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		target := fields[1]
		if target == root || strings.HasPrefix(target, root+"/") {
			targets = append(targets, target)
		}
	}

	sort.Slice(targets, func(i, j int) bool {
		return strings.Count(targets[i], "/") > strings.Count(targets[j], "/")
	})

	for _, target := range targets {
		log.Debugf("unmounting '%s'", target)
		if err := syscall.Unmount(target, syscall.MNT_DETACH); err != nil {
			log.Errorf("failed to un-mount '%s': %s", target, err)
		}
	}

	return nil
}
