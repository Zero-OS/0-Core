package main

import (
	"fmt"
	"os"

	"github.com/g8os/core0/base"
	"github.com/g8os/core0/base/logger"
	"github.com/g8os/core0/base/pm"
	pmcore "github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/settings"
	"github.com/g8os/core0/coreX/bootstrap"
	"github.com/g8os/core0/coreX/options"
	"github.com/op/go-logging"

	_ "github.com/g8os/core0/base/builtin"
	_ "github.com/g8os/core0/coreX/builtin"
	"os/signal"
	"syscall"
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
		log.Infof("Job result for command '%s' is '%s'", cmd, result.State)
	})

	mgr.Run()

	bs := bootstrap.NewBootstrap()

	handleSignal(bs)

	if err := bs.Bootstrap(opt.Hostname()); err != nil {
		log.Fatalf("Failed to bootstrap corex: %s", err)
	}

	sinkID := fmt.Sprintf("%d", opt.CoreID())

	sinkCfg := settings.SinkConfig{
		URL:      fmt.Sprintf("redis://%s", opt.RedisSocket()),
		Password: opt.RedisPassword(),
	}

	cl, err := core.NewSinkClient(&sinkCfg, sinkID, opt.ReplyTo())
	if err != nil {
		log.Fatal("Failed to get connection to redis at %s", sinkCfg.URL)
	}

	sinks := map[string]core.SinkClient{
		"main": cl,
	}

	log.Infof("Configure redis logger")

	mgr.AddMessageHandler(rl.Log)

	//forward stats messages to core0
	mgr.AddStatsHandler(func(op, key string, value float64, tags string) {
		fmt.Printf("10::core-%d.%s:%f|%s|%s\n", opt.CoreID(), key, value, op, tags)
	})

	//start jobs sinks.
	core.StartSinks(pm.GetManager(), sinks)

	//wait
	select {}
}
