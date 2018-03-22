// +build arm
// CC=armv6j-hardfloat-linux-gnueabi-gcc CGO_ENABLED=1 CGO_LDFLAGS="-Wl,-rpath -Wl,/lib" GOOS=linux GOARCH=arm GOARM=6 go build

package kvm

import (
	"github.com/op/go-logging"
	"github.com/zero-os/0-core/apps/core0/screen"
	"github.com/zero-os/0-core/apps/core0/subsys/containers"
)

var (
	log = logging.MustGetLogger("kvm")
)

func KVMSubsystem(conmgr containers.ContainerManager, cell *screen.RowCell) error {
	log.Errorf("kvm disabled on arm")
	return nil
}

