package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/op/go-logging"
	"github.com/zero-os/0-core/apps/core0/options"
)

const (
	LogPath = "/var/log/core.log"
)

var (
	output *os.File
)

/*
Logs are written to stdout if not `quiet`
Stderr is redirected to a `log` file
Stderr is written to by default (if redirection succeeded)

Note, in case of panic, the panic will only show up in the logs
if only `debug` flag is set. since this will open the log file
in sync mode
*/

func redirect(name string) (err error) {
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if options.Options.Kernel.Is("debug") {
		flags |= os.O_SYNC
	}

	output, err = os.OpenFile(name, flags, 0600)
	if err != nil {
		return err
	}
	os.Stderr.Close()
	return syscall.Dup2(int(output.Fd()), 2)
}

//setLogging stdout logs to file
func setLogging(p string) error {
	var backends []logging.Backend
	if !options.Options.Kernel.Is("quiet") {
		normal := logging.NewLogBackend(os.Stdout, "", 0)
		backends = append(backends, normal)
	}

	if err := redirect(p); err == nil {
		backends = append(backends,
			logging.NewLogBackend(output, "", 0),
		)
	}

	level := logging.GetLevel("")
	logging.SetBackend(backends...)
	logging.SetLevel(level, "")
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

	return setLogging(p)
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
