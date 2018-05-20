package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/op/go-logging"
	"github.com/zero-os/0-core/apps/core0/options"
	"github.com/zero-os/0-core/base/pm"
)

const (
	//LogPath main log file path
	LogPath = "/var/log/core.log"
	//CrashPath crash report
	CrashPath = "/var/log/crash"
)

var (
	logf *os.File
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
	f, err := os.OpenFile(
		name,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_SYNC,
		0600,
	)

	if err != nil {
		return err
	}

	defer f.Close()
	return syscall.Dup2(int(f.Fd()), 2)
}

//setLogging stdout logs to file
func setLogging(main string, crash string) error {
	if err := redirect(crash); err != nil {
		return err
	}

	var backends []logging.Backend
	if !options.Options.Kernel.Is("quiet") {
		normal := logging.NewLogBackend(os.Stdout, "", 0)
		backends = append(backends, normal)
	}
	var err error
	logf, err = os.Create(main)
	if err != nil {
		return err
	}

	backends = append(backends, logging.NewLogBackend(logf, "", 0))

	level := logging.GetLevel("")
	logging.SetBackend(backends...)
	logging.SetLevel(level, "")
	return nil
}

//Rotate logs
func Rotate() error {
	if logf != nil {
		logf.Close()
		os.Rename(
			logf.Name(),
			fmt.Sprintf("%s.%v", logf.Name(), time.Now().Format("20060102-150405")),
		)
	}

	pm.Kill("syslogd") // make sure we also restart syslogd (in case /var/log) has changed.
	return setLogging(LogPath, CrashPath)
}

//HandleRotation force log rotation on
func HandleRotation() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGUSR1)
	go func() {
		for range ch {
			if err := Rotate(); err != nil {
				log.Errorf("failed to rotate logs")
			}
		}
	}()
}
