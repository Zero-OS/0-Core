package kvm

import (
	"encoding/json"
	"github.com/google/shlex"
	"github.com/libvirt/libvirt-go"
	"github.com/zero-os/0-core/base/pm/process"
	"github.com/zero-os/0-core/base/pm/stream"
	"runtime/debug"
	"strings"
)

func (m *kvmManager) handle(conn *libvirt.Connect, domain *libvirt.Domain, event *libvirt.DomainEventLifecycle) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("error processing domain event: %v", err)
			log.Error(string(debug.Stack()))
		}
	}()

	uuid, _ := domain.GetUUIDString()
	parts, _ := shlex.Split(event.String())
	data := map[string]interface{}{
		"uuid": uuid,
	}

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			data[kv[0]] = kv[1]
		}
	}

	m.evch <- data
}

func (m *kvmManager) events(ctx *process.Context) (interface{}, error) {
	var sequence uint64 = 1
	for data := range m.evch {
		data["sequence"] = sequence
		bytes, _ := json.Marshal(data)
		ctx.Log(string(bytes), stream.LevelResultJSON)

		sequence++
	}

	return nil, nil
}
