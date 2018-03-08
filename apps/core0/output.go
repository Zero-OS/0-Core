package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/op/go-logging"
	"github.com/zero-os/0-core/core0/options"
)

const (
	LogPath = "/var/log/core.log"
)

var (
	output *os.File
)

//Redirect stdout logs to file
func Redirect(p string) error {
	var backends []logging.Backend
	if !options.Options.Kernel.Is("quiet") {
		normal := logging.NewLogBackend(os.Stderr, "", 0)
		backends = append(backends, normal)
	}

	if log, err := os.Create(p); err == nil {
		output = log
		backends = append(backends,
			logging.NewLogBackend(log, "", 0),
		)
	}

	logging.SetBackend(backends...)
	return nil
}

//Rotate logs
func Rotate(p string) error {
	if output != nil {
		output.Close()
		os.Rename(
			output.Name(),
			fmt.Sprintf("%s.%v", output.Name(), time.Now().Format("20060102-150405")),
		)
	}

	return Redirect(p)
}

//HandleRotation force log rotation on
func HandleRotation() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGUSR1)
	go func() {
		for range ch {
			if err := Rotate(LogPath); err != nil {
				log.Errorf("failed to rotate logs")
			}
		}
	}()
}
