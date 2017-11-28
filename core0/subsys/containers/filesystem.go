package containers

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/disk"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/settings"
	"github.com/zero-os/0-core/core0/helper"
)

const (
	BackendBaseDir       = "/var/cache/containers"
	ContainerBaseRootDir = "/mnt"
)

func (c *container) name() string {
	return fmt.Sprintf("container-%d", c.id)
}

func (c *container) mountFList(src string, target string, hooks ...pm.RunnerHook) error {
	//check
	namespace := fmt.Sprintf("containers/%s", c.name())
	storage := c.Args.Storage
	if storage == "" {
		storage = settings.Settings.Globals.Get("storage", "ardb://hub.gig.tech:16379")
		c.Args.Storage = storage
	}

	return helper.MountFList(namespace, storage, src, target, hooks...)
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
	//mount root flist.
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

	if err := c.mountFList(c.Args.Root, root, onSBExit); err != nil {
		return fmt.Errorf("mount-root-flist(%s)", err)
	}

	for src, dst := range c.Args.Mount {
		target := path.Join(root, dst)
		if err := os.MkdirAll(target, 0755); err != nil {
			return fmt.Errorf("mkdirAll(%s)", err)
		}
		//src can either be a location on HD, or another flist
		u, err := url.Parse(src)
		if err != nil {
			return fmt.Errorf("bad mount source '%s': %s", src, err)
		}

		if u.Scheme == "" {
			if err := syscall.Mount(src, target, "", syscall.MS_BIND, ""); err != nil {
				return fmt.Errorf("mount-bind(%s)", err)
			}
		} else {
			//assume a flist
			if err := c.mountFList(src, target); err != nil {
				return fmt.Errorf("mount-bind-flist(%s)", err)
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

	return syscall.Mount(coreXSrc, coreXTarget, "", syscall.MS_BIND, "")
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
