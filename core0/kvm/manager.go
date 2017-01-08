package kvm

import (
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/process"
	"github.com/g8os/core0/base/pm/core"
)

type kvmManager struct{}

const (
	kvmCreateCommand = "kvm.create"
)
func init() {
	mgr := kvmManager{}

	pm.CmdMap[kvmCreateCommand] = process.NewInternalProcessFactory(mgr.create)
}

func (m *kvmManager) create(cmd *core.Command) (interface{}, error) {

	return nil, nil
}