package main

import (
	"fmt"
	"os"
	"time"

	"github.com/g8os/core0/base"
	"github.com/g8os/core0/base/pm"
	pmcore "github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/settings"
	"github.com/g8os/core0/core0/assets"
	"github.com/g8os/core0/core0/bootstrap"
	"github.com/g8os/core0/core0/logger"
	"github.com/g8os/core0/core0/options"
	"github.com/g8os/core0/core0/screen"
	"github.com/g8os/core0/core0/stats"
	"github.com/g8os/core0/core0/subsys/containers"
	"github.com/g8os/core0/core0/subsys/kvm"
	"github.com/op/go-logging"

	_ "github.com/g8os/core0/base/builtin"
	_ "github.com/g8os/core0/core0/builtin"
	_ "github.com/g8os/core0/core0/builtin/btrfs"
	"github.com/g8os/core0/core0/transport"
	"os/signal"
	"syscall"
)

var (
	log = logging.MustGetLogger("main")
)

func init() {
	signal.Ignore(syscall.SIGABRT, syscall.SIGHUP, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
	formatter := logging.MustStringFormatter("%{time}: %{color}%{module} %{level:.1s} > %{message} %{color:reset}")
	logging.SetFormatter(formatter)
}

func main() {
	var options = options.Options
	fmt.Println(core.Version())
	if options.Version() {
		os.Exit(0)
	}

	if err := Redirect(LogPath); err != nil {
		log.Errorf("failed to redirect output streams: %s", err)
	}

	HandleRotation()

	if err := screen.New(2); err != nil {
		log.Critical(err)
	}

	screen.Push(&screen.CenteredText{
		TextSection: screen.TextSection{
			Attributes: screen.Attributes{screen.Bold},
			Text:       string(assets.MustAsset("text/logo.txt")),
		},
	})
	screen.Push(&screen.TextSection{})
	screen.Push(&screen.TextSection{
		Attributes: screen.Attributes{screen.Green},
		Text:       core.Version().Short(),
	})
	screen.Push(&screen.TextSection{})
	screen.Refresh()

	if err := settings.LoadSettings(options.Config()); err != nil {
		log.Fatal(err)
	}

	if errors := settings.Settings.Validate(); len(errors) > 0 {
		for _, err := range errors {
			log.Errorf("%s", err)
		}

		log.Fatalf("\nConfig validation error, please fix and try again.")
	}

	var config = settings.Settings

	var loglevel string
	if options.Kernel.Is("verbose") {
		loglevel = "DEBUG"
	} else {
		loglevel = config.Main.LogLevel
	}

	level, err := logging.LogLevel(loglevel)
	if err != nil {
		log.Fatal("invalid log level: %s", loglevel)
	}

	logging.SetLevel(level, "")

	pm.InitProcessManager(config.Main.MaxJobs)

	//start process mgr.
	log.Infof("Starting process manager")
	mgr := pm.GetManager()

	mgr.AddResultHandler(func(cmd *pmcore.Command, result *pmcore.JobResult) {
		log.Debugf("Job result for command '%s' is '%s'", cmd, result.State)
	})

	mgr.Run()

	//configure logging handlers from configurations
	log.Infof("Configure logging")
	logger.InitLogging()

	bs := bootstrap.NewBootstrap()
	bs.Bootstrap()

	log.Infof("Setting up stats aggregator clients")
	if config.Stats.Redis.Enabled {
		aggregator, err := stats.NewRedisStatsAggregator(config.Stats.Redis.Address, "", 1000, time.Duration(config.Stats.Redis.FlushInterval)*time.Second)
		if err != nil {
			log.Errorf("failed to initialize redis stats aggregator: %s", err)
		} else {
			mgr.AddStatsHandler(aggregator.Aggregate)
		}
	}

	screen.Push(&screen.SplitterSection{Title: "System Information"})

	row := &screen.RowSection{
		Cells: make([]screen.RowCell, 2),
	}
	screen.Push(row)

	sink, err := transport.NewSink(mgr, transport.SinkConfig{Port: 6379})
	if err != nil {
		log.Errorf("failed to start command sink: %s", err)
	}

	contMgr, err := containers.ContainerSubsystem(sink, &row.Cells[0])
	if err != nil {
		log.Fatal("failed to intialize container subsystem", err)
	}

	if err := kvm.KVMSubsystem(contMgr, &row.Cells[1]); err != nil {
		log.Errorf("failed to initialize kvm subsystem", err)
	}

	log.Infof("Starting local transport")
	local, err := NewLocal(contMgr, "/var/run/core.sock")
	if err != nil {
		log.Errorf("Failed to start local transport: %s", err)
	} else {
		local.Start()
	}

	//start jobs sinks.
	log.Infof("Starting Sinks")

	sink.Start()
	screen.Refresh()

	//wait
	select {}
}
