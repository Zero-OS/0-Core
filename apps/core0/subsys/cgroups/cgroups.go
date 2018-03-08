package cgroups

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"syscall"
)

type mkg func(name, subsys string) Group

type Group interface {
	Name() string
	Subsystem() string
	Task(pid int) error
}

const (
	DevicesSubsystem = "devices"
	CGroupBase       = "/sys/fs/cgroup"
)

var (
	once       sync.Once
	subsystems = map[string]mkg{
		DevicesSubsystem: mkDevicesGroup,
	}
)

func Init() (err error) {
	once.Do(func() {
		os.MkdirAll(CGroupBase, 0755)
		err = syscall.Mount("cgroup_root", CGroupBase, "tmpfs", 0, "")
		if err != nil {
			return
		}

		for sub := range subsystems {
			p := path.Join(CGroupBase, sub)
			os.MkdirAll(p, 0755)

			err = syscall.Mount(sub, p, "cgroup", 0, sub)
			if err != nil {
				return
			}
		}
	})

	return
}

func GetGroup(name string, subsystem string) (Group, error) {
	mkg, ok := subsystems[subsystem]
	if !ok {
		return nil, fmt.Errorf("unknown subsystem '%s'", subsystem)
	}

	p := path.Join(CGroupBase, subsystem, name)
	if err := os.Mkdir(p, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}

	return mkg(name, subsystem), nil
}

type cgroup struct {
	name   string
	subsys string
}

func (g *cgroup) Name() string {
	return g.name
}

func (g *cgroup) Subsystem() string {
	return g.subsys
}

func (g *cgroup) base() string {
	return path.Join(CGroupBase, g.subsys, g.name)
}

func (g *cgroup) Task(pid int) error {
	return ioutil.WriteFile(path.Join(g.base(), "cgroup.procs"), []byte(fmt.Sprint(pid)), 0644)
}
