package main

import (
	"fmt"
	"github.com/g8os/core0/base"
	"github.com/g8os/core0/base/logger"
	"github.com/g8os/core0/base/pm"
	pmcore "github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/settings"
	"github.com/g8os/core0/coreX/bootstrap"
	"github.com/g8os/core0/coreX/options"
	"github.com/op/go-logging"
	"os"

	_ "github.com/g8os/core0/base/builtin"
	_ "github.com/g8os/core0/coreX/builtin"
)

var (
	log = logging.MustGetLogger("main")
)

func init() {
	formatter := logging.MustStringFormatter("%{color}%{module} %{level:.1s} > %{message} %{color:reset}")
	logging.SetFormatter(formatter)
}

func main() {
	if errors := options.Options.Validate(); len(errors) != 0 {
		for _, err := range errors {
			log.Errorf("Validation Error: %s\n", err)
		}

		os.Exit(1)
	}

	var opt = options.Options

	pm.InitProcessManager(opt.MaxJobs())

	//start process mgr.
	log.Infof("Starting process manager")
	mgr := pm.GetManager()

	//handle process results. Forwards the result to the correct controller.
	mgr.AddResultHandler(func(cmd *pmcore.Command, result *pmcore.JobResult) {
		log.Infof("Job result for command '%s' is '%s'", cmd, result.State)
	})

	mgr.Run()

	//start local transport
	log.Infof("Starting local transport")
	local, err := core.NewLocal("/var/run/core.sock")
	if err != nil {
		log.Errorf("Failed to start local transport: %s", err)
	} else {
		go local.Serve()
	}

	bs := bootstrap.NewBootstrap()
	if err := bs.Bootstrap(); err != nil {
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
	rl := logger.NewRedisLogger(opt.CoreID(), opt.RedisSocket(), "",
		[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, 100000)
	mgr.AddMessageHandler(rl.Log)
	//
	//log.Infof("Setting up stats buffers")
	//if config.Stats.Redis.Enabled {
	//	redis := core.NewRedisStatsBuffer(config.Stats.Redis.Address, "", 1000, time.Duration(config.Stats.Redis.FlushInterval)*time.Millisecond)
	//	mgr.AddStatsFlushHandler(redis.Handler)
	//}

	//start jobs sinks.
	core.StartSinks(pm.GetManager(), sinks)

	//wait
	select {}
}
