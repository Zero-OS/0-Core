package transport

import (
	"fmt"
	"github.com/siddontang/ledisdb/config"
	"github.com/siddontang/ledisdb/ledis"
	"github.com/siddontang/ledisdb/server"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/core0/assets"
	"github.com/zero-os/0-core/core0/options"
	"time"
)

const (
	SinkRoute = core.Route("sink")
	SinkQueue = "core:default"
	DBIndex   = 0
)

type Sink struct {
	mgr    *pm.PM
	ch     *channel
	server *server.App
	db     *ledis.DB
}

type SinkConfig struct {
	Port int
}

func (c *SinkConfig) Local() string {
	return fmt.Sprintf("127.0.0.1:%d", c.Port)
}

func NewSink(mgr *pm.PM, c SinkConfig) (*Sink, error) {
	cfg := config.NewConfigDefault()
	cfg.DBName = "memory"
	cfg.DataDir = "/var/core0"
	cfg.Addr = fmt.Sprintf(":%d", c.Port)
	if orgs, ok := options.Options.Kernel.Get("organization"); ok {
		org := orgs[len(orgs)-1]
		auth, err := AuthMethod(org, string(assets.MustAsset("text/itsyouonline.pub")))
		if err != nil {
			return nil, err
		}
		cfg.AuthMethod = auth
	}

	crt, key, err := generateCRT()
	if err != nil {
		return nil, err
	}

	cfg.TLS = config.TLS{
		Enabled:     true,
		Certificate: crt,
		Key:         key,
	}

	server, err := server.NewApp(cfg)
	if err != nil {
		return nil, err
	}

	db, err := server.Ledis().Select(DBIndex)
	if err != nil {
		return nil, err
	}

	sink := &Sink{
		mgr:    mgr,
		server: server,
		db:     db,
		ch:     newChannel(db),
	}

	return sink, nil
}

func (sink *Sink) DB() *ledis.DB {
	return sink.db
}

func (sink *Sink) handlePublic(cmd *core.Command, result *core.JobResult) {
	if err := sink.Forward(result); err != nil {
		log.Errorf("failed to forward result: %s", cmd.ID)
	}
}

func (sink *Sink) process() {
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
	if result.State != core.StateDuplicateID {
		/*
			Client tried to push a command with a duplicate id, it means another job
			is running with that ID so we shouldn't flag
		*/
		sink.ch.UnFlag(result.ID)
	}
	return sink.ch.Respond(result)
}

func (sink *Sink) Flag(id string) error {
	return sink.ch.Flag(id)
}

func (sink *Sink) Start() {
	go sink.server.Run()
	go sink.process()
}

func (sink *Sink) Result(job string, timeout int) (*core.JobResult, error) {
	if sink.ch.Flagged(job) {
		return sink.ch.GetResponse(job, timeout)
	} else {
		return nil, fmt.Errorf("unknown job id '%s' (may be it has expired)", job)
	}
}
