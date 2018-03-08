package main

import (
	"fmt"
	"os"

	"github.com/op/go-logging"
	"github.com/zero-os/0-core/base"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/apps/coreX/bootstrap"
	"github.com/zero-os/0-core/apps/coreX/options"

	"os/signal"
	"syscall"

	"encoding/json"
	_ "github.com/zero-os/0-core/base/builtin"
	_ "github.com/zero-os/0-core/apps/coreX/builtin"
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

	pm.MaxJobs = opt.MaxJobs()
	pm.New()

	input := os.NewFile(3, "|input")
	output := os.NewFile(4, "|output")

	dispatcher := NewDispatcher(output)

	//start process mgr.
	log.Infof("Starting process manager")

	pm.AddHandle(dispatcher)
	pm.Start()

	bs := bootstrap.NewBootstrap()

	if err := bs.Bootstrap(opt.Hostname()); err != nil {
		log.Fatalf("Failed to bootstrap corex: %s", err)
	}

	handleSignal(bs)

	dec := json.NewDecoder(input)
	for {
		var cmd pm.Command
		if err := dec.Decode(&cmd); err != nil {
			log.Errorf("failed to decode command message: %s", err)

		}

		_, err := pm.Run(&cmd)

		if err == pm.UnknownCommandErr {
			result := pm.NewJobResult(&cmd)
			result.State = pm.StateUnknownCmd
			dispatcher.Result(&cmd, result)
		} else if err != nil {
			log.Errorf("unknown error while queueing command: %s", err)
		}
	}
}
