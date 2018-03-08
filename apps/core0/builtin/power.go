package builtin

import (
	"syscall"

	"github.com/zero-os/0-core/base/pm"
)

const (
	cmdReboot   = "core.reboot"
	cmdPowerOff = "core.poweroff"
)

func init() {
	pm.RegisterBuiltIn(cmdReboot, restart)
	pm.RegisterBuiltIn(cmdPowerOff, poweroff)
}

func restart(cmd *pm.Command) (interface{}, error) {
	pm.Killall()
	syscall.Sync()
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	return nil, nil
}

func poweroff(cmd *pm.Command) (interface{}, error) {
	pm.Killall()
	syscall.Sync()
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	return nil, nil
}
