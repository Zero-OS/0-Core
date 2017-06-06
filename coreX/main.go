package main

import (
	"fmt"
	"os"

	"github.com/op/go-logging"
	"github.com/zero-os/0-core/base"
	"github.com/zero-os/0-core/base/pm"
	pmcore "github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/coreX/bootstrap"
	"github.com/zero-os/0-core/coreX/options"

	"os/signal"
	"syscall"

	"encoding/json"
	_ "github.com/zero-os/0-core/base/builtin"
	_ "github.com/zero-os/0-core/coreX/builtin"
)

var (
	log = logging.MustGetLogger("main")
)

func init() {
	formatter := logging.MustStringFormatter("%{color}%{module} %{level:.1s} > %{message} %{color:reset}")
	logging.SetFormatter(formatter)
	logging.SetLevel(logging.DEBUG, "")
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

	pm.InitProcessManager(opt.MaxJobs())

	input := os.NewFile(3, "|input")
	output := os.NewFile(4, "|output")

	dispatcher := NewDispatcher(output)

	//start process mgr.
	log.Infof("Starting process manager")
	mgr := pm.GetManager()

	mgr.AddResultHandler(dispatcher.Result)
	mgr.AddMessageHandler(dispatcher.Message)
	mgr.AddStatsHandler(dispatcher.Stats)

	mgr.Run()

	bs := bootstrap.NewBootstrap()

	if err := bs.Bootstrap(opt.Hostname()); err != nil {
		log.Fatalf("Failed to bootstrap corex: %s", err)
	}

	handleSignal(bs)

	dec := json.NewDecoder(input)
	for {
		var cmd pmcore.Command
		if err := dec.Decode(&cmd); err != nil {
			log.Errorf("failed to decode command message: %s", err)

		}
		mgr.PushCmd(&cmd)
	}
}
