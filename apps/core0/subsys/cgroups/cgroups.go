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

type mkg func(name string, subsys Subsystem) Group

//Group generic cgroup interface
type Group interface {
	Name() string
	Subsystem() Subsystem
	Task(pid int) error
	Tasks() ([]int, error)
	Root() Group
	Reset()
}

type Subsystem string

const (
	//DevicesSubsystem device subsystem
	DevicesSubsystem = Subsystem("devices")
	//CPUSetSubsystem cpu subsystem
	CPUSetSubsystem = Subsystem("cpuset")
	//MemorySubsystem memory subsystem
	MemorySubsystem = Subsystem("memory")

	//CGroupBase base mount point
	CGroupBase = "/sys/fs/cgroup"
)

var (
	log        = logging.MustGetLogger("cgroups")
	once       sync.Once
	subsystems = map[Subsystem]mkg{
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
			p := path.Join(CGroupBase, string(sub))
			os.MkdirAll(p, 0755)

			err = syscall.Mount(string(sub), p, "cgroup", 0, string(sub))
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

		pm.RegisterBuiltIn("cgroup.reset", reset)

		pm.RegisterBuiltIn("cgroup.cpuset.spec", cpusetSpec)
		pm.RegisterBuiltIn("cgroup.memory.spec", memorySpec)
	})

	return
}

//GetGroup creaes a group if it does not exist
func GetGroup(subsystem Subsystem, name string) (Group, error) {
	mkg, ok := subsystems[subsystem]
	if !ok {
		return nil, fmt.Errorf("unknown subsystem '%s'", subsystem)
	}

	p := path.Join(CGroupBase, string(subsystem), name)
	if err := os.Mkdir(p, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}

	return mkg(name, subsystem), nil
}

//Get group only if it exists
func Get(subsystem Subsystem, name string) (Group, error) {
	if !Exists(subsystem, name) {
		return nil, ErrDoesNotExist
	}

	return GetGroup(subsystem, name)
}

//GetGroups gets all the available groups names grouped by susbsytem
func GetGroups() (map[Subsystem][]string, error) {
	result := make(map[Subsystem][]string)
	for sub := range subsystems {
		// skip devices subsystem (only cpuset and memory)
		if sub == DevicesSubsystem {
			continue
		}
		info, err := ioutil.ReadDir(path.Join(CGroupBase, string(sub)))
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
func Remove(subsystem Subsystem, name string) error {
	if !Exists(subsystem, name) {
		return nil
	}

	builder := subsystems[subsystem]
	group := builder(name, subsystem)
	tasks, err := group.Tasks()
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		return os.Remove(path.Join(CGroupBase, string(subsystem), name))
	}

	root := group.Root()
	for _, task := range tasks {
		root.Task(task)
	}

	return os.Remove(path.Join(CGroupBase, string(subsystem), name))
}

//Exists Check if a cgroup exists
func Exists(subsystem Subsystem, name string) bool {
	_, ok := subsystems[subsystem]
	if !ok {
		return false
	}

	p := path.Join(CGroupBase, string(subsystem), name)
	info, err := os.Stat(p)
	if err != nil {
		return false
	}

	return info.IsDir()
}

type cgroup struct {
	name   string
	subsys Subsystem
}

func (g *cgroup) Name() string {
	return g.name
}

func (g *cgroup) Subsystem() Subsystem {
	return g.subsys
}

func (g *cgroup) base() string {
	return path.Join(CGroupBase, string(g.subsys), g.name)
}

func (g *cgroup) Task(pid int) error {
	return ioutil.WriteFile(path.Join(g.base(), "cgroup.procs"), []byte(fmt.Sprint(pid)), 0644)
}

func (g *cgroup) Tasks() ([]int, error) {
	raw, err := ioutil.ReadFile(path.Join(g.base(), "cgroup.procs"))
	if err != nil {
		return nil, err
	}

	pids := make([]int, 0)
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
