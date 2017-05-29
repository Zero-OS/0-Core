package containers

import (
	"encoding/json"
	"fmt"
	blog "github.com/zero-os/0-core/base/logger"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/stream"
	"github.com/zero-os/0-core/core0/logger"
	"github.com/zero-os/0-core/core0/stats"
	"io"
)

type Message struct {
	Type    string          `json:"type"`
	Command string          `json:"command"`
	Payload json.RawMessage `json:"payload"`
}

func (c *container) forward() {
	enc := json.NewEncoder(c.channel)
	for cmd := range c.forwardChan {
		if err := enc.Encode(cmd); err != nil {
			log.Errorf("failed to forward command (%s) to container (%d)", cmd.ID, c.id)
		}
	}
}

func (c *container) rewind() {
	decoder := json.NewDecoder(c.channel)
	for {

		var message Message
		err := decoder.Decode(&message)
		if err == io.EOF {
			return
		} else if err != nil {
			log.Errorf("failed to process corex %d message: %s", c.id, err)
			return
		}

		switch message.Type {
		case "result":
			var result core.JobResult
			if err := json.Unmarshal(message.Payload, &result); err != nil {
				log.Errorf("failed to load container command result: %s", err)
			}
			result.Container = uint64(c.id)
			c.mgr.sink.Forward(&result)
		case "log":
			var msg stream.Message
			if err := json.Unmarshal(message.Payload, &msg); err != nil {
				log.Errorf("failed to load container log message: %s", err)
			}

			logger.Current.LogRecord(&blog.LogRecord{
				Core:    c.id,
				Command: message.Command,
				Message: &msg,
			})
		case "stats":
			var stat stats.Stats
			if err := json.Unmarshal(message.Payload, &stat); err != nil {
				log.Errorf("failed to load container stat message: %s", err)
			}
			//push stats to aggregation system
			pm.GetManager().Aggregate(string(stat.Operation), fmt.Sprintf("core-%d.%s", c.id, stat.Key), stat.Value, stat.Tags)
		default:
			log.Warningf("got unknown message type from container(%d): %s", c.id, message.Type)
		}
	}
}
