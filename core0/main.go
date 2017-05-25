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
	formatter := logging.MustStringFormatter("%{time}: %{color}%{module} %{level:.1s} > %{message} %{color:reset}")
	logging.SetFormatter(formatter)

	//we don't use signal.Ignore because the Ignore is actually inherited by children
	//even after execve which makes all child process not exit when u send them a sigterm or sighup
	signal.Notify(make(chan os.Signal), syscall.SIGABRT, syscall.SIGHUP, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
}

func main() {
	var options = options.Options
	fmt.Println(core.Version())
	if options.Version() {
		os.Exit(0)
	}

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

	if err := Redirect(LogPath); err != nil {
		log.Errorf("failed to redirect output streams: %s", err)
	}

	HandleRotation()

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
	cfg := transport.SinkConfig{Port: 6379}
	sink, err := transport.NewSink(mgr, cfg)
	if err != nil {
		log.Errorf("failed to start command sink: %s", err)
	}

	logger.ConfigureLogging(sink.DB())

	bs := bootstrap.NewBootstrap()
	bs.Bootstrap()

	screen.Push(&screen.SplitterSection{Title: "System Information"})

	row := &screen.RowSection{
		Cells: make([]screen.RowCell, 2),
	}
	screen.Push(row)

	contMgr, err := containers.ContainerSubsystem(sink, &row.Cells[0])
	if err != nil {
		log.Fatal("failed to intialize container subsystem", err)
	}

	if err := kvm.KVMSubsystem(sink, contMgr, &row.Cells[1]); err != nil {
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

	if config.Stats.Enabled {
		aggregator, err := stats.NewRedisStatsAggregator(cfg.Local(), "", 1000, time.Duration(config.Stats.FlushInterval)*time.Second)
		if err != nil {
			log.Errorf("failed to initialize redis stats aggregator: %s", err)
		} else {
			mgr.AddStatsHandler(aggregator.Aggregate)
		}
	}

	//wait
	select {}
}
