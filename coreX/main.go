package main

import (
	"fmt"
	"os"

	"github.com/g8os/core0/base"
	"github.com/g8os/core0/base/logger"
	"github.com/g8os/core0/base/pm"
	pmcore "github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/coreX/bootstrap"
	"github.com/g8os/core0/coreX/options"
	"github.com/op/go-logging"

	"os/signal"
	"syscall"

	_ "github.com/g8os/core0/base/builtin"
	_ "github.com/g8os/core0/coreX/builtin"
)

var (
	log = logging.MustGetLogger("main")
)

func init() {
	formatter := logging.MustStringFormatter("%{color}%{module} %{level:.1s} > %{message} %{color:reset}")
	logging.SetFormatter(formatter)
	logging.SetLevel(logging.INFO, "")
}

func handleSignal(bs *bootstrap.Bootstrap) {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGTERM)
	go func(ch <-chan os.Signal, bs *bootstrap.Bootstrap) {
		<-ch
		log.Infof("Received SIGTERM, terminating.")
		bs.UnBootstrap()
		os.Exit(0)
	}(ch, bs)
}

func main() {
	var opt = options.Options
	fmt.Println(core.Version())
	if opt.Version() {
		os.Exit(0)
	}

	if errors := options.Options.Validate(); len(errors) != 0 {
		for _, err := range errors {
			log.Errorf("Validation Error: %s\n", err)
		}

		os.Exit(1)
	}

	//redis logger for commands
	rl := logger.NewRedisLogger(uint16(opt.CoreID()), opt.RedisSocket(), "", nil, 100000)

	//set backend so coreX logs itself also get pushed to redis
	logging.SetBackend(
		&logBackend{
			logger: rl,
			cmd: pmcore.Command{
				ID: "core-x",
			},
		},
	)

	pm.InitProcessManager(opt.MaxJobs())

	//start process mgr.
	log.Infof("Starting process manager")
	mgr := pm.GetManager()

	//handle process results. Forwards the result to the correct controller.
	mgr.AddResultHandler(func(cmd *pmcore.Command, result *pmcore.JobResult) {
		result.Container = opt.CoreID()
		log.Debugf("Job result for command '%s' is '%s'", cmd, result.State)
	})

	mgr.Run()

	bs := bootstrap.NewBootstrap()

	if opt.Unprivileged() {
		mgr.SetUnprivileged()
	}

	if err := bs.Bootstrap(opt.Hostname()); err != nil {
		log.Fatalf("Failed to bootstrap corex: %s", err)
	}

	handleSignal(bs)

	log.Infof("Configure redis logger")

	mgr.AddMessageHandler(rl.Log)

	//forward stats messages to core0
	mgr.AddStatsHandler(func(op, key string, value float64, tags string) {
		fmt.Printf("10::core-%d.%s:%f|%s|%s\n", opt.CoreID(), key, value, op, tags)
	})

	sinkID := fmt.Sprintf("%d", opt.CoreID())

	sink, err := core.NewSink(sinkID, mgr, core.SinkConfig{URL: fmt.Sprintf("redis://%s", opt.RedisSocket())})
	if err != nil {
		log.Errorf("failed to start command sink: %s", err)
	}
	sink.Start()

	//wait
	select {}
}
