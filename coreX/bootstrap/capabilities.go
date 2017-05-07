package bootstrap

// #cgo LDFLAGS: -lcap
// #include <sys/capability.h>
import "C"
import (
	"fmt"
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

	if C.cap_set_proc(cap) != 0 {
		return fmt.Errorf("failed to set capabilities")
	}

	return nil
}
