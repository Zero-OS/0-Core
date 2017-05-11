package transport

import (
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"time"
)

const (
	SinkRoute = core.Route("sink")
	SinkQueue = "core:default"
)

type Sink struct {
	mgr *pm.PM
	ch  *channel
}

type SinkConfig struct {
	URL      string `json:"url"`
	Password string `json:"password"`
}

func NewSink(mgr *pm.PM, config SinkConfig) (*Sink, error) {
	public, err := newChannel(config.URL, config.Password)
	if err != nil {
		return nil, err
	}

	sink := &Sink{
		mgr: mgr,
		ch:  public,
	}

	return sink, nil
}

func (sink *Sink) handlePublic(cmd *core.Command, result *core.JobResult) {
	if err := sink.Forward(result); err != nil {
		log.Errorf("failed to forward result: %s", cmd.ID)
	}
}

func (sink *Sink) run() {
	sink.mgr.AddRouteResultHandler(SinkRoute, sink.handlePublic)

	for {
		var command core.Command
		err := sink.ch.GetNext(SinkQueue, &command)
		if err != nil {
			log.Errorf("Failed to get next command from (%s): %s", SinkQueue, err)
			<-time.After(200 * time.Millisecond)
			continue
		}

		if command.ID == "" {
			log.Warningf("receiving a command with no ID, dropping")
			continue
		}

		sink.ch.Flag(command.ID)
		command.Route = SinkRoute
		log.Debugf("Starting command %s", &command)

		sink.mgr.PushCmd(&command)
	}
}

func (sink *Sink) Forward(result *core.JobResult) error {
	sink.ch.UnFlag(result.ID)
	return sink.ch.Respond(result)
}

func (sink *Sink) Flag(id string) error {
	return sink.ch.Flag(id)
}

func (sink *Sink) Start() {
	go sink.run()
}

func (sink *Sink) Result(job string, timeout int) (*core.JobResult, error) {
	if sink.ch.Flagged(job) {
		return sink.ch.GetResponse(job, timeout)
	} else {
		return nil, fmt.Errorf("unknown job id '%s' (may be it has expired)", job)
	}
}
