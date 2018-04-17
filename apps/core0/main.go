package main

import (
	"fmt"
	"os"

	"github.com/op/go-logging"
	"github.com/zero-os/0-core/apps/core0/assets"
	"github.com/zero-os/0-core/apps/core0/bootstrap"
	"github.com/zero-os/0-core/apps/core0/logger"
	"github.com/zero-os/0-core/apps/core0/options"
	"github.com/zero-os/0-core/apps/core0/screen"
	"github.com/zero-os/0-core/apps/core0/stats"
	"github.com/zero-os/0-core/apps/core0/subsys/containers"
	"github.com/zero-os/0-core/apps/core0/subsys/kvm"
	"github.com/zero-os/0-core/base"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/settings"

	"os/signal"
	"syscall"

	_ "github.com/zero-os/0-core/apps/core0/builtin"
	_ "github.com/zero-os/0-core/apps/core0/builtin/btrfs"
	"github.com/zero-os/0-core/apps/core0/transport"
	_ "github.com/zero-os/0-core/base/builtin"
)

var (
	log = logging.MustGetLogger("main")
)

func init() {
	formatter := logging.MustStringFormatter("%{time}: %{color}%{module} %{level:.1s} > %{message} %{color:reset}")
	logging.SetFormatter(formatter)

	//we don't use signal.Ignore because the Ignore is actually inherited by children
	//even after execve which makes all child process not exit when u send them a sigterm or sighup
	var signals []os.Signal
	for i := 1; i <= 31; i++ {
		if syscall.Signal(i) == syscall.SIGUSR1 ||
			syscall.Signal(i) == syscall.SIGCHLD {
			continue
		}
		signals = append(signals, syscall.Signal(i))
	}

	signal.Notify(make(chan os.Signal), signals...)
}

//Splash setup splash screen
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
		Text: fmt.Sprintf("Boot Params: %s",
			options.Options.Kernel.String("debug", "organization", "zerotier", "quiet", "development"), //flags we care about
		),
	})
	screen.Push(&screen.TextSection{})
	screen.Push(&screen.TextSection{
		Text: "[Alt+F1: Kernel logs view] [Alt+F2: This screen]",
	})
	screen.Push(&screen.TextSection{})
	screen.Refresh()
}

type console struct{}

func (*console) Result(cmd *pm.Command, result *pm.JobResult) {
	log.Debugf("Job result for command '%s' is '%s'", cmd, result.State)
}

func main() {
	var options = options.Options
	fmt.Println(core.Version())
	if options.Version() {
		os.Exit(0)
	}

	if !options.Agent() {
		//Only allow splash screen if debug is not set, or if not running in agent mode
		Splash()
	}

	if err := settings.LoadSettings(options.Config()); err != nil {
		log.Fatal(err)
	}

	if errors := settings.Settings.Validate(); len(errors) > 0 {
		for _, err := range errors {
			log.Errorf("%s", err)
		}

		log.Fatalf("\nConfig validation error, please fix and try again.")
	}

	if !options.Agent() {
		//Logging prepration
		if err := Rotate(); err != nil {
			log.Errorf("failed to setup logging: %s", err)
		}

		//Handle log rotation requests (SIGNALS)
		HandleRotation()
	}

	level := logging.INFO
	if options.Kernel.Is("debug") {
		level = logging.DEBUG
	}

	logging.SetLevel(level, "")

	var config = settings.Settings

	pm.MaxJobs = config.Main.MaxJobs

	pm.New()

	//start process mgr.
	log.Infof("Starting process manager")

	pm.AddHandle((*console)(nil))
	pm.Start()

	//configure logging handlers from configurations
	log.Infof("Configure logging")
	cfg := transport.SinkConfig{Port: 6379}
	sink, err := transport.NewSink(cfg)
	if err != nil {
		log.Errorf("failed to start command sink: %s", err)
	}

	logger.ConfigureLogging(sink)

	bs := bootstrap.NewBootstrap(options.Agent())
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

	if err := kvm.KVMSubsystem(contMgr, &row.Cells[1]); err != nil {
		log.Errorf("failed to initialize kvm subsystem: %s", err)
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
		aggregator := stats.NewLedisStatsAggregator(sink)
		pm.AddHandle(aggregator)
	}

	select {}
}
