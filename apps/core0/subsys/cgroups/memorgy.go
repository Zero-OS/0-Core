package cgroups

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
)

type MemoryGroup interface {
	Group
	Limits() (int, int, error)
	Limit(mem, swap int) error
}

func mkMemoryGroup(name string, subsys Subsystem) Group {
	return &memoryCGroup{
		cgroup{name: name, subsys: subsys},
	}
}

type memoryCGroup struct {
	cgroup
}

func (c *memoryCGroup) memFile() string {
	return path.Join(c.base(), "memory.limit_in_bytes")
}

func (c *memoryCGroup) swapFile() string {
	return path.Join(c.base(), "memory.memsw.limit_in_bytes")
}

//Limits returns mem, swap, err
func (c *memoryCGroup) Limits() (int, int, error) {
	var values [2]int
	for i, p := range []string{c.memFile(), c.swapFile()} {
		data, err := ioutil.ReadFile(p)
		if err != nil {
			return 0, 0, err
		}

		if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &values[i]); err != nil {
			return 0, 0, err
		}
	}

	return values[0], values[1], nil
}

func (c *memoryCGroup) set(name string, value int) error {
	return ioutil.WriteFile(name, []byte(fmt.Sprintf("%d", value)), 0644)
}

func (c *memoryCGroup) Reset() {
	c.set(c.swapFile(), -1)
	c.set(c.memFile(), -1)
}

func (c *memoryCGroup) Limit(mem, swap int) error {
	//write to those files are tricket because at any moment in time we can't have memsw less than memory
	//so we need to know the current values. then
	cmem, cswap, err := c.Limits()
	if err != nil {
		return err
	}

	if mem == 0 {
		//updating swap only
		mem = cmem
	}

	swap = mem + swap

	if cswap < mem {
		//if current swap is less than mem, we need to set swap first
		c.set(c.swapFile(), swap)
		c.set(c.memFile(), mem)
	} else {
		//if new memory is equal or less than currnt swap, we set memory first
		c.set(c.memFile(), mem)
		c.set(c.swapFile(), swap)
	}

	return nil
}

func (c *memoryCGroup) Root() Group {
	return &memoryCGroup{
		cgroup: cgroup{subsys: c.subsys},
	}
}

var _ MemoryGroup = &memoryCGroup{}
