package bootstrap

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("bootstrap")
)

type Bootstrap struct {
}

func NewBootstrap() *Bootstrap {
	return &Bootstrap{}
}

//Bootstrap registers extensions and startup system services.
func (b *Bootstrap) Bootstrap(hostname string) error {
	log.Infof("Mounting proc")
	if err := syscall.Mount("none", "/proc", "proc", 0, ""); err != nil {
		return err
	}

	if err := syscall.Mount("none", "/dev", "devtmpfs", 0, ""); err != nil {
		return err
	}

	if err := syscall.Mount("none", "/dev/pts", "devpts", 0, ""); err != nil {
		return err
	}

	if err := updateHostname(hostname); err != nil {
		return err
	}

	return nil
}

func updateHostname(hostname string) error {
	log.Infof("Set hostname to %s", hostname)

	// update /etc/hostname
	fHostname, err := os.OpenFile("/etc/hostname", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer fHostname.Close()
	fmt.Fprint(fHostname, hostname)

	// update /etc/hosts
	fHosts, err := os.OpenFile("/etc/hosts", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer fHosts.Close()
	fmt.Fprintf(fHosts, "127.0.0.1    %s.local %s\n", hostname, hostname)
	fmt.Fprint(fHosts, "127.0.0.1    localhost.localdomain localhost\n")

	// call hostname command
	cmd := exec.Command("hostname", hostname)
	return cmd.Run()
}
