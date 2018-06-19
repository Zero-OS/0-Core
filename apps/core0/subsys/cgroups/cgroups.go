package cgroups

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"

	logging "github.com/op/go-logging"
	"github.com/zero-os/0-core/base/pm"
)

type mkg func(name, subsys string) Group

//Group generic cgroup interface
type Group interface {
	Name() string
	Subsystem() string
	Task(pid int) error
	Tasks() ([]int, error)
	Root() Group
	Reset()
}

const (
	//DevicesSubsystem device subsystem
	DevicesSubsystem = "devices"
	//CPUSetSubsystem cpu subsystem
	CPUSetSubsystem = "cpuset"
	//MemorySubsystem memory subsystem
	MemorySubsystem = "memory"

	//CGroupBase base mount point
	CGroupBase = "/sys/fs/cgroup"
)

var (
	log        = logging.MustGetLogger("cgroups")
	once       sync.Once
	subsystems = map[string]mkg{
		DevicesSubsystem: mkDevicesGroup,
		CPUSetSubsystem:  mkCPUSetGroup,
		MemorySubsystem:  mkMemoryGroup,
	}

	//ErrDoesNotExist does not exist error
	ErrDoesNotExist = fmt.Errorf("cgroup does not exist")
	//ErrInvalidType invalid cgroup type
	ErrInvalidType = fmt.Errorf("cgroup of invalid type")
)

//Init Initialized the cgroup subsystem
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

		pm.RegisterBuiltIn("cgroup.list", list)
		pm.RegisterBuiltIn("cgroup.ensure", ensure)
		pm.RegisterBuiltIn("cgroup.remove", remove)

		pm.RegisterBuiltIn("cgroup.tasks", tasks)
		pm.RegisterBuiltIn("cgroup.task-add", taskAdd)
		pm.RegisterBuiltIn("cgroup.task-remove", taskRemove)

		pm.RegisterBuiltIn("cgroup.cpuset.reset", cpusetReset)
		pm.RegisterBuiltIn("cgroup.cpuset.spec", cpusetSpec)

		pm.RegisterBuiltIn("cgroup.memory.reset", memoryReset)
		pm.RegisterBuiltIn("cgroup.memory.spec", memorySpec)
	})

	return
}

//GetGroup creaes a group if it does not exist
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

//Get group only if it exists
func Get(name, subsystem string) (Group, error) {
	if !Exists(name, subsystem) {
		return nil, ErrDoesNotExist
	}

	return GetGroup(name, subsystem)
}

//GetGroups gets all the available groups names grouped by susbsytem
func GetGroups() (map[string][]string, error) {
	result := make(map[string][]string)
	for sub := range subsystems {
		info, err := ioutil.ReadDir(path.Join(CGroupBase, sub))
		if err != nil {
			return nil, err
		}
		for _, dir := range info {
			if !dir.IsDir() {
				continue
			}

			result[sub] = append(result[sub], dir.Name())
		}
	}

	return result, nil
}

//Remove removes a cgroup
func Remove(name, subsystem string) error {
	if !Exists(name, subsystem) {
		return nil
	}

	builder := subsystems[subsystem]
	group := builder(name, subsystem)
	tasks, err := group.Tasks()
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		return os.Remove(path.Join(CGroupBase, subsystem, name))
	}

	root := group.Root()
	for _, task := range tasks {
		root.Task(task)
	}

	return os.Remove(path.Join(CGroupBase, subsystem, name))
}

//Exists Check if a cgroup exists
func Exists(name, subsystem string) bool {
	_, ok := subsystems[subsystem]
	if !ok {
		return false
	}

	p := path.Join(CGroupBase, subsystem, name)
	info, err := os.Stat(p)
	if err != nil {
		return false
	}

	return info.IsDir()
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

func (g *cgroup) Tasks() ([]int, error) {
	raw, err := ioutil.ReadFile(path.Join(g.base(), "cgroup.procs"))
	if err != nil {
		return nil, err
	}

	var pids []int
	for _, s := range strings.Split(string(raw), "\n") {
		if len(s) == 0 {
			continue
		}
		var pid int
		if _, err := fmt.Sscanf(s, "%d", &pid); err != nil {
			return nil, err
		}
		pids = append(pids, pid)
	}

	return pids, nil
}
