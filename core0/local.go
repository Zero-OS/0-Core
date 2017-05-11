package main

import (
	"encoding/json"
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/utils"
	"github.com/g8os/core0/core0/subsys/containers"
	"net"
	"os"
	"strconv"
	"strings"
)

type Local struct {
	listener *net.UnixListener
	mgr      containers.ContainerManager
}

type LocalCmd struct {
	Sync      bool            `json:"sync"`
	Container string          `json:"container"`
	Content   json.RawMessage `json:"content"`
}

func NewLocal(mgr containers.ContainerManager, s string) (*Local, error) {
	if utils.Exists(s) {
		os.Remove(s)
	}

	addr, err := net.ResolveUnixAddr("unix", s)
	if err != nil {
		return nil, err
	}
	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, err
	}
	return &Local{
		mgr:      mgr,
		listener: listener,
	}, nil
}

func (l *Local) container(x string) containers.Container {
	if x == "" {
		return nil
	}

	id, err := strconv.ParseUint(x, 10, 16)
	if err == nil {
		return l.mgr.Of(uint16(id))
	}

	tags := strings.Split(x, ",")
	return l.mgr.GetOneWithTags(tags...)
}

func (l *Local) server(con net.Conn) {
	//read command
	job := &core.JobResult{
		State: core.StateError,
	}

	defer func() {
		//send result
		m, _ := json.Marshal(job)
		if _, err := con.Write(m); err != nil {
			log.Errorf("Failed to write response to local transport: %s", err)
		}
		con.Close()
	}()

	decoder := json.NewDecoder(con)
	var lcmd LocalCmd

	if err := decoder.Decode(&lcmd); err != nil {
		job.Streams = []string{"", fmt.Sprintf("Failed to decode message: %s", err)}
		return
	}

	cmd, err := core.LoadCmd(lcmd.Content)
	if err != nil {
		job.Streams = []string{"", fmt.Sprintf("Failed to extract command: %s", err)}
		return
	}

	container := l.container(lcmd.Container)

	if lcmd.Container != "" && container == nil {
		job.Streams = []string{"", fmt.Sprintf("couldn't match any containers with '%s'", lcmd.Container)}
		return
	}

	if container == nil {
		runner, err := pm.GetManager().RunCmd(cmd)
		if err != nil {
			job.Streams = []string{"", fmt.Sprintf("Failed to get job runner for command(%s): %s", cmd.Command, err)}
			return
		}

		if lcmd.Sync {
			job = runner.Wait()
		}

		return
	} else {
		contjob, err := l.mgr.Dispatch(container.ID(), cmd)
		if err != nil {
			job.Streams = []string{"", fmt.Sprintf("Failed to dispatch command (%s): %s", cmd.Command, err)}
			return
		}
		job = contjob
	}
}

func (l *Local) start() {
	defer l.listener.Close()
	for {
		con, err := l.listener.Accept()
		if err != nil {
			log.Errorf("local transport error: %s", err)
		}
		go l.server(con)
	}
}

func (l *Local) Start() {
	go l.start()
}
