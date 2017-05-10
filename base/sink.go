package core

import (
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"time"
)

const (
	SinkRoute = core.Route("sink")
)

type Sink struct {
	key string
	mgr *pm.PM
	ch  *channel
}

type SinkConfig struct {
	URL      string `json:"url"`
	Password string `json:"password"`
}

func NewSink(key string, mgr *pm.PM, config SinkConfig) (*Sink, error) {
	public, err := newChannel(config.URL, config.Password)
	if err != nil {
		return nil, err
	}

	sink := &Sink{
		key: key,
		mgr: mgr,
		ch:  public,
	}

	return sink, nil
}

func (sink *Sink) DefaultQueue() string {
	return fmt.Sprintf("core:%v",
		sink.key,
	)
}

func (sink *Sink) handlePublic(cmd *core.Command, result *core.JobResult) {
	//yes, we unflag the command on the private redis not the public, it's were we
	//keep the flags.
	sink.ch.UnFlag(cmd.ID)
	if err := sink.ch.Respond(result); err != nil {
		log.Errorf("Failed to respond to command %s: %s", cmd, err)
	}
}

func (sink *Sink) run() {
	sink.mgr.AddRouteResultHandler(SinkRoute, sink.handlePublic)

	queue := sink.DefaultQueue()
	for {
		var command core.Command
		err := sink.ch.GetNext(queue, &command)
		if err != nil {
			log.Errorf("Failed to get next command from %s(%s): %s", sink.key, queue, err)
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

func (sink *Sink) Forward(queue string, cmd *core.Command) error {
	defer sink.ch.Flag(cmd.ID)
	return sink.ch.Push(queue, cmd)
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
