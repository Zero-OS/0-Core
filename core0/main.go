package main

import (
	"fmt"
	"os"

	"github.com/op/go-logging"
	"github.com/zero-os/0-core/base"
	"github.com/zero-os/0-core/base/pm"
	pmcore "github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/settings"
	"github.com/zero-os/0-core/core0/assets"
	"github.com/zero-os/0-core/core0/bootstrap"
	"github.com/zero-os/0-core/core0/logger"
	"github.com/zero-os/0-core/core0/options"
	"github.com/zero-os/0-core/core0/screen"
	"github.com/zero-os/0-core/core0/stats"
	"github.com/zero-os/0-core/core0/subsys/containers"
	"github.com/zero-os/0-core/core0/subsys/kvm"

	_ "github.com/zero-os/0-core/base/builtin"
	_ "github.com/zero-os/0-core/core0/builtin"
	_ "github.com/zero-os/0-core/core0/builtin/btrfs"
	"github.com/zero-os/0-core/core0/subsys/apps"
	"github.com/zero-os/0-core/core0/transport"
	"os/signal"
	"path"
	"strings"
	"syscall"
)

var (
	log = logging.MustGetLogger("main")
)

func init() {
	formatter := logging.MustStringFormatter("%{time}: %{color}%{module} %{level:.1s} > %{message} %{color:reset}")
	logging.SetFormatter(formatter)

	normal := logging.NewLogBackend(os.Stderr, "", 0)

	backends := []logging.Backend{normal}

	if !options.Options.Kernel.Is("quiet") {
		opts, _ := options.Options.Kernel.Get("console")
		for _, opt := range opts {
			console := strings.SplitN(opt, ",", 2)[0]

			out, err := os.OpenFile(path.Join("/dev", console), syscall.O_WRONLY|syscall.O_NOCTTY, 0644)
			if err != nil {
				fmt.Println("failed to redirect logs to console")
				continue
			}

			backends = append(backends,
				logging.NewLogBackend(out, "", 0),
			)
		}
	}

	logging.SetBackend(backends...)
	level := logging.INFO
	if options.Options.Kernel.Is("debug") {
		level = logging.DEBUG
	}

	logging.SetLevel(level, "")

	//we don't use signal.Ignore because the Ignore is actually inherited by children
	//even after execve which makes all child process not exit when u send them a sigterm or sighup
	signal.Notify(make(chan os.Signal), syscall.SIGABRT, syscall.SIGHUP, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
}

func Splash() {

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
		Text:       fmt.Sprintf("Version: %s", core.Version().Short()),
	})
	screen.Push(&screen.TextSection{
		Attributes: screen.Attributes{screen.Bold},
		Text: fmt.Sprintf("Params: %s",
			options.Options.Kernel.String("debug", "organization", "zerotier", "quiet"), //flags we care about
		),
	})
	screen.Push(&screen.TextSection{})
	screen.Refresh()
}

func main() {
	var options = options.Options
	fmt.Println(core.Version())
	if options.Version() {
		os.Exit(0)
	}

	Splash()

	if err := settings.LoadSettings(options.Config()); err != nil {
		log.Fatal(err)
	}

	if errors := settings.Settings.Validate(); len(errors) > 0 {
		for _, err := range errors {
			log.Errorf("%s", err)
		}

		log.Fatalf("\nConfig validation error, please fix and try again.")
	}

	//Redirect the stdout, and stderr so we make sure we don't lose crashes that terminates
	//the process.
	if err := Redirect(LogPath); err != nil {
		log.Errorf("failed to redirect output streams: %s", err)
	}

	HandleRotation()

	var config = settings.Settings

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
	bs.First()

	screen.Push(&screen.TextSection{})
	screen.Push(&screen.SplitterSection{Title: "System Information"})

	row := &screen.RowSection{
		Cells: make([]screen.RowCell, 2),
	}
	screen.Push(row)

	contMgr, err := containers.ContainerSubsystem(sink, &row.Cells[0])
	if err != nil {
		log.Fatal("failed to intialize container subsystem", err)
	}

	bs.Second()
	if err := apps.AppsSubsystem(sink); err != nil {
		log.Errorf("failed to initialize apps subsystem")
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

	if config.Stats.Enabled {
		aggregator := stats.NewLedisStatsAggregator(sink.DB())
		mgr.AddStatsHandler(aggregator.Aggregate)
	}

	//wait
	select {}
}
