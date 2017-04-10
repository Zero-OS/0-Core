package bootstrap

import (
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/settings"
	"github.com/g8os/core0/base/utils"
	"github.com/op/go-logging"
	"github.com/shirou/gopsutil/process"
	"github.com/vishvananda/netlink"
	"os"
	"syscall"
)

var (
	log = logging.MustGetLogger("bootstrap")
)

type Bootstrap struct {
}

func NewBootstrap() *Bootstrap {
	return &Bootstrap{}
}

func (b *Bootstrap) setupLO() {
	//we don't crash on lo device setup because this will fail anyway if the container runs
	//with host_networking.
	//plus setting up the lo device should not stop the container from starting if it failed anyway

	link, err := netlink.LinkByName("lo")
	if err != nil {
		log.Warningf("failed to get lo device")
		return
	}

	addr, _ := netlink.ParseAddr("127.0.0.1/8")
	if err := netlink.AddrAdd(link, addr); err != nil {
		log.Warningf("failed to setup lo address: %s", err)
	}

	addr, _ = netlink.ParseAddr("::1/128")
	if err := netlink.AddrAdd(link, addr); err != nil {
		log.Warningf("failed to setup lo address: %s", err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		log.Warningf("failed to bring lo interface up: %s", err)
	}
}

func (o *Bootstrap) setupFS() error {
	os.MkdirAll("/etc", 0755)
	os.MkdirAll("/var/run", 0755)

	os.MkdirAll("/sys", 0755)
	if err := syscall.Mount("none", "/sys", "sysfs", 0, ""); err != nil {
		return err
	}

	os.MkdirAll("/proc", 0755)
	if err := syscall.Mount("none", "/proc", "proc", 0, ""); err != nil {
		return err
	}

	os.MkdirAll("/dev/pts", 0755)
	if err := syscall.Mount("none", "/dev", "devtmpfs", 0, ""); err != nil {
		return err
	}

	if err := syscall.Mount("none", "/dev/pts", "devpts", 0, ""); err != nil {
		return err
	}

	return nil
}

func (b *Bootstrap) startup() error {
	var included settings.IncludedSettings
	if err := utils.LoadTomlFile("/.startup.toml", &included); err != nil {
		return err
	}

	tree, errs := included.GetStartupTree()
	if errs != nil {
		return fmt.Errorf("failed to build startup tree: %v", errs)
	}

	pm.GetManager().RunSlice(tree.Slice(settings.AfterInit.Weight(), settings.ToTheEnd.Weight()))

	return nil
}

//Bootstrap registers extensions and startup system services.
func (b *Bootstrap) Bootstrap(hostname string) error {
	log.Debugf("setting up lo device")
	b.setupLO()

	log.Debugf("setting up mounts")
	if err := b.setupFS(); err != nil {
		return err
	}

	log.Debugf("setting up hostname")
	if err := updateHostname(hostname); err != nil {
		return err
	}

	log.Debugf("linkin mtab")
	if err := linkMtab(); err != nil {
		return err
	}

	log.Debugf("startup services")

	if err := b.plugins(); err != nil {
		log.Errorf("failed to load plugins: %s", err)
	}

	if err := b.startup(); err != nil {
		log.Errorf("failed to startup container services: %s", err)
	}

	return nil
}

func (b *Bootstrap) UnBootstrap() {
	//clean up behind (kill all processes)
	pids, _ := process.Pids()
	//kill all children.
	for _, pid := range pids {
		if pid == 1 {
			continue
		}
		log.Infof("stopping PID: %d", pid)
		ps, err := process.NewProcess(pid)

		if err != nil {
			log.Errorf("failed to kill pid (%d): %s", pid, err)
			continue
		}

		if err := ps.Kill(); err != nil {
			log.Errorf("failed to kill pid (%d): %s", pid, err)
			continue
		}
	}

	for _, mnt := range []string{"/dev/pts", "/dev", "proc"} {
		log.Infof("Unmounting: %s", mnt)
		if err := syscall.Unmount(mnt, syscall.MNT_DETACH); err != nil {
			log.Errorf("failed to unmount %s", err)
		}
	}
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

	return syscall.Sethostname([]byte(hostname))
}

func linkMtab() error {
	return os.Symlink("../proc/self/mounts", "/etc/mtab")
}
