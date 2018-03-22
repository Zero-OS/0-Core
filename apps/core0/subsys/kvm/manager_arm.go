// +build arm

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

