package cgroups

import (
	"io/ioutil"
	"path"
	"strings"
)

type CPUSetGroup interface {
	Group
	Set(sepc string) error
	Get() (string, error)
}

func mkCPUSetGroup(name, subsys string) Group {
	return &devicesCGroup{
		cgroup{name: name, subsys: subsys},
	}
}

type cpusetCGroup struct {
	cgroup
}

func (c *cpusetCGroup) Set(spec string) error {
	if err := ioutil.WriteFile(path.Join(c.base(), "cpuset.cpus"), []byte(spec), 0644); err != nil {
		return err
	}

	if err := ioutil.WriteFile(path.Join(c.base(), "cpuset.mems"), []byte(spec), 0644); err != nil {
		return err
	}

	return nil
}

func (c *cpusetCGroup) Get() (string, error) {
	data, err := ioutil.ReadFile(path.Join(c.base(), "cpuset.cpus"))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func (c *cpusetCGroup) Root() Group {
	return &cpusetCGroup{
		cgroup: cgroup{subsys: c.subsys},
	}
}
