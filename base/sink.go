package core

import (
	"time"

	"encoding/json"
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/g8os/core0/base/settings"
)

const (
	PublicRoute  = core.Route("public")
	PrivateRoute = core.Route("private")
)

type Sink struct {
	key     string
	mgr     *pm.PM
	public  *channel
	private *channel
}

func NewSink(key string, mgr *pm.PM, config *settings.Sink) (*Sink, error) {
	public, err := newChannel(&config.Public)
	if err != nil {
		return nil, err
	}

	private, err := newChannel(&config.Private)
	if err != nil {
		return nil, err
	}

	sink := &Sink{
		key:     key,
		mgr:     mgr,
		public:  public,
		private: private,
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
	sink.private.UnFlag(cmd.ID)
	if err := sink.public.Respond(result); err != nil {
		log.Errorf("Failed to respond to command %s: %s", cmd, err)
	}
}

func (sink *Sink) handlePrivate(cmd *core.Command, result *core.JobResult) {
	sink.private.UnFlag(cmd.ID)
	if err := sink.private.Respond(result); err != nil {
		log.Errorf("Failed to respond to command %s: %s", cmd, err)
	}
}

func (sink *Sink) run() {
	sink.mgr.AddRouteResultHandler(PublicRoute, sink.handlePublic)
	sink.mgr.AddRouteResultHandler(PrivateRoute, sink.handlePrivate)

	queue := sink.DefaultQueue()
	for {
		var command core.Command
		err := sink.public.GetNext(queue, &command)
		if err != nil {
			log.Errorf("Failed to get next command from %s(%s): %s", sink.key, queue, err)
			<-time.After(200 * time.Millisecond)
			continue
		}

		sink.private.Flag(command.ID)
		command.Route = PrivateRoute
		log.Debugf("Starting command %s", &command)

		sink.mgr.PushCmd(&command)
	}
}

func (sink *Sink) Start() {
	go sink.run()
}

func (sink *Sink) Result(job string, timeout int) (*core.JobResult, error) {
	if sink.private.Flagged(job) {
		return sink.private.GetResponse(job, timeout)
	} else {
		return nil, fmt.Errorf("unknown job id '%s' (may be it has expired)", job)
	}
}

func (sink *Sink) getResult(cmd *core.Command) (interface{}, error) {
	cmd.Route = PublicRoute
	var args struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	return sink.Result(args.ID, cmd.MaxTime)
}

func (sink *Sink) StartResponder() {
	pm.CmdMap["core.result"] = process.NewInternalProcessFactory(sink.getResult)
}
