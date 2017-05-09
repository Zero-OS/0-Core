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

//
//type Channel interface {
//	GetNext(queue string, command *core.Command) error
//	Respond(result *core.JobResult) error
//}

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
	if err := sink.public.Respond(result); err != nil {
		log.Errorf("Failed to respond to command %s: %s", cmd, err)
	}
}

func (sink *Sink) handlePrivate(cmd *core.Command, result *core.JobResult) {
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

		command.Route = PrivateRoute

		log.Debugf("Starting command %s", &command)

		sink.mgr.PushCmd(&command)
	}
}

func (sink *Sink) Start() {
	go sink.run()
}

func (sink *Sink) getResult(cmd *core.Command) (interface{}, error) {
	cmd.Route = PublicRoute
	var args struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	log.Debugf("Getting result for '%s'", args.ID)

	//push result from private to public
	return sink.private.GetResponse(args.ID, 10)
}

func (sink *Sink) StartResponder() {
	pm.CmdMap["core.result"] = process.NewInternalProcessFactory(sink.getResult)
}
