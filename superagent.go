package main

import (
	"github.com/g8os/core.base"
	"github.com/g8os/core.base/logger"
	"github.com/g8os/core.base/pm"
	pmcore "github.com/g8os/core.base/pm/core"
	"github.com/g8os/core.base/settings"
	"github.com/g8os/core0/bootstrap"
	"github.com/op/go-logging"
	"time"

	"fmt"
	_ "github.com/g8os/core.base/builtin"
	_ "github.com/g8os/core0/builtin"
	"github.com/g8os/core0/containers"
	"github.com/g8os/core0/options"
)

var (
	log = logging.MustGetLogger("main")
)

func init() {
	formatter := logging.MustStringFormatter("%{color}%{module} %{level:.1s} > %{message} %{color:reset}")
	logging.SetFormatter(formatter)
}

func main() {
	var options = options.Options

	if err := settings.LoadSettings(options.Config()); err != nil {
		log.Fatal(err)
	}

	if errors := settings.Settings.Validate(); len(errors) > 0 {
		for _, err := range errors {
			log.Errorf("%s", err)
		}

		log.Fatalf("\nConfig validation error, please fix and try again.")
	}

	if settings.Settings.Sink == nil {
		settings.Settings.Sink = make(map[string]settings.SinkConfig)
	}

	var config = settings.Settings

	pm.InitProcessManager(config.Main.MaxJobs)

	//start process mgr.
	log.Infof("Starting process manager")
	mgr := pm.GetManager()

	//handle process results. Forwards the result to the correct controller.
	mgr.AddResultHandler(func(cmd *pmcore.Command, result *pmcore.JobResult) {
		log.Infof("Job result for command '%s' is '%s'", cmd, result.State)
	})

	mgr.Run()

	//configure logging handlers from configurations
	log.Infof("Configure logging")
	logger.ConfigureLogging(uint64(0))

	//start local transport
	log.Infof("Starting local transport")
	local, err := core.NewLocal("/var/run/core.sock")
	if err != nil {
		log.Errorf("Failed to start local transport: %s", err)
	} else {
		go local.Serve()
	}

	bs := bootstrap.NewBootstrap()
	bs.Bootstrap()

	sinkID := fmt.Sprintf("default")

	//build list with ACs that we will poll from.
	sinks := make(map[string]core.SinkClient)
	for key, sinkCfg := range config.Sink {
		cl, err := core.NewSinkClient(&sinkCfg, sinkID)
		if err != nil {
			log.Warning("Can't reach sink %s: %s", sinkCfg.URL, err)
			continue
		}

		sinks[key] = cl
	}

	log.Infof("Setting up stats buffers")
	if config.Stats.Redis.Enabled {
		redis := core.NewRedisStatsBuffer(config.Stats.Redis.Address, "", 1000, time.Duration(config.Stats.Redis.FlushInterval)*time.Millisecond)
		mgr.AddStatsFlushHandler(redis.Handler)
	}

	//start/register containers commands and process
	containers.Containers(sinks)

	//start jobs sinks.
	log.Infof("Starting Sinks")
	core.StartSinks(pm.GetManager(), sinks)

	//wait
	select {}
}
