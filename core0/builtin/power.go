package builtin

import (
	"syscall"

	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
)

const (
	cmdReboot   = "core.reboot"
	cmdPowerOff = "core.poweroff"
)

func init() {
	pm.CmdMap[cmdReboot] = process.NewInternalProcessFactory(restart)
	pm.CmdMap[cmdPowerOff] = process.NewInternalProcessFactory(poweroff)
}

func restart(cmd *core.Command) (interface{}, error) {
	pm.GetManager().Killall()
	syscall.Sync()
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	return nil, nil
}

func poweroff(cmd *core.Command) (interface{}, error) {
	pm.GetManager().Killall()
	syscall.Sync()
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	return nil, nil
}
