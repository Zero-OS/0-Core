package main

import (
	"encoding/json"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/stream"
	"os"
	"sync"
)

type MessageType string

const (
	ResultMessage MessageType = "result"
	LogMessage    MessageType = "log"
	StatsMessage  MessageType = "stats"
)

type Message struct {
	Type    MessageType `json:"type"`
	Command string      `json:"command"`
	Payload interface{} `json:"payload"`
}
type Dispatcher struct {
	enc *json.Encoder
	m   sync.Mutex
}

func NewDispatcher(out *os.File) *Dispatcher {
	return &Dispatcher{enc: json.NewEncoder(out)}
}

func (d *Dispatcher) Pre(cmd *pm.Command) {
	//force all coreX children to be in the same group
	cmd.Flags.NoSetPGID = true
}

func (d *Dispatcher) Result(cmd *pm.Command, result *pm.JobResult) {
	d.m.Lock()
	defer d.m.Unlock()

	d.enc.Encode(Message{Type: ResultMessage, Command: cmd.ID, Payload: result})
}

func (d *Dispatcher) Message(cmd *pm.Command, msg *stream.Message) {
	d.m.Lock()
	defer d.m.Unlock()

	d.enc.Encode(Message{Type: LogMessage, Command: cmd.ID, Payload: msg})
}

func (d *Dispatcher) Stats(operation string, key string, value float64, id string, tags ...pm.Tag) {
	d.m.Lock()
	defer d.m.Unlock()

	d.enc.Encode(Message{Type: StatsMessage, Payload: map[string]interface{}{
		"operation": operation,
		"key":       key,
		"value":     value,
		"tags":      tags,
	}})
}
