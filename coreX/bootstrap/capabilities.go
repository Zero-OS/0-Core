package bootstrap

// #cgo LDFLAGS: -lcap
// #include <sys/capability.h>
import "C"
import (
	"fmt"
	"syscall"
	"unsafe"
)

func (b *Bootstrap) revokePrivileges() error {
	cap := C.cap_init()
	defer C.cap_free(unsafe.Pointer(cap))

	if C.cap_clear(cap) != 0 {
		return fmt.Errorf("failed to clear up capabilities")
	}

	flags := []C.cap_value_t{
		C.CAP_SETPCAP,
		C.CAP_MKNOD,
		C.CAP_AUDIT_WRITE,
		C.CAP_CHOWN,
		C.CAP_NET_RAW,
		C.CAP_DAC_OVERRIDE,
		C.CAP_FOWNER,
		C.CAP_FSETID,
		C.CAP_KILL,
		C.CAP_SETGID,
		C.CAP_SETUID,
		C.CAP_NET_BIND_SERVICE,
		C.CAP_SYS_CHROOT,
		C.CAP_SETFCAP,
	}

	if C.cap_set_flag(cap, C.CAP_PERMITTED, C.int(len(flags)), &flags[0], C.CAP_SET) != 0 {
		return fmt.Errorf("failed to set capabiliteis flags (perm)")
	}
	if C.cap_set_flag(cap, C.CAP_EFFECTIVE, C.int(len(flags)), &flags[0], C.CAP_SET) != 0 {
		return fmt.Errorf("failed to set capabiliteis flags (effective)")
	}
	if C.cap_set_flag(cap, C.CAP_INHERITABLE, C.int(len(flags)), &flags[0], C.CAP_SET) != 0 {
		return fmt.Errorf("failed to set capabiliteis flags (inheritable)")
	}

	//drop bounding set for children.
	bound := []uintptr{
		C.CAP_SYS_MODULE,
		C.CAP_SYS_RAWIO,
		C.CAP_SYS_PACCT,
		C.CAP_SYS_ADMIN,
		C.CAP_SYS_NICE,
		C.CAP_SYS_RESOURCE,
		C.CAP_SYS_TIME,
		C.CAP_SYS_TTY_CONFIG,
		C.CAP_AUDIT_CONTROL,
		C.CAP_MAC_OVERRIDE,
		C.CAP_MAC_ADMIN,
		C.CAP_NET_ADMIN,
		C.CAP_SYSLOG,
		C.CAP_DAC_READ_SEARCH,
		C.CAP_LINUX_IMMUTABLE,
		C.CAP_NET_BROADCAST,
		C.CAP_IPC_LOCK,
		C.CAP_IPC_OWNER,
		C.CAP_SYS_PTRACE,
		C.CAP_SYS_BOOT,
		C.CAP_LEASE,
		C.CAP_WAKE_ALARM,
		C.CAP_BLOCK_SUSPEND,
	}

	for _, c := range bound {
		if _, _, err := syscall.Syscall6(syscall.SYS_PRCTL, syscall.PR_CAPBSET_DROP,
			c, 0, 0, 0, 0); err != 0 {
			return fmt.Errorf("failed to lower cap bound: %s", err)
		}
	}

	if C.cap_set_proc(cap) != 0 {
		return fmt.Errorf("failed to set capabilities")
	}

	return nil
}
