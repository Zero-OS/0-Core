package main

import (
	"encoding/json"
	"github.com/codegangsta/cli"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/pborman/uuid"
	"net"
	"sync"
)

type Command struct {
	Sync      bool         `json:"sync"`
	Container string       `json:"container"`
	Content   core.Command `json:"content"`
}

type TransportOptions struct {
	Timeout int
	ID      string
}

type Transport interface {
	Run(cmd Command) (*Response, error)
}

type unixSocketTransport struct {
	con *net.UnixConn
	opt *TransportOptions
	m   sync.Mutex
}

func NewUnixSocketTransport(n string, opt *TransportOptions) (Transport, error) {
	addr, err := net.ResolveUnixAddr("unix", n)
	if err != nil {
		return nil, err
	}

	con, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, err
	}

	return &unixSocketTransport{
		con: con,
		opt: opt,
	}, nil
}

func NewTransport(c *cli.Context) (Transport, error) {
	return NewUnixSocketTransport(c.GlobalString("socket"), &TransportOptions{
		Timeout: c.GlobalInt("timeout"),
		ID:      c.GlobalString("id"),
	})
}

func (t *unixSocketTransport) setDefaults(cmd *Command) error {
	if t.opt == nil {
		return nil
	}

	cmd.Content.MaxTime = t.opt.Timeout
	return nil
}

func (t *unixSocketTransport) Run(cmd Command) (*Response, error) {
	t.m.Lock()
	defer t.m.Unlock()

	if t.opt.ID == "" {
		cmd.Content.ID = uuid.New()
	} else {
		cmd.Content.ID = t.opt.ID
	}

	if err := t.setDefaults(&cmd); err != nil {
		return nil, err
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	if _, err := t.con.Write(data); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(t.con)
	var response Response
	if err := decoder.Decode(&response); err != nil {
		return nil, err
	}

	response.ID = cmd.Content.ID
	return &response, nil
}

func WithTransport(action func(t Transport, c *cli.Context)) cli.ActionFunc {
	return func(c *cli.Context) error {
		t, err := NewTransport(c)
		if err != nil {
			log.Fatalf("failed to connect to g8os manager: %s", err)
		}

		action(t, c)
		return nil
	}
}
