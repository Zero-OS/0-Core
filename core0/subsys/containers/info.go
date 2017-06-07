package containers

import (
	"encoding/json"
	"fmt"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
)

func (m *containerManager) ztInfo(cmd *core.Command) (interface{}, error) {
	var args ContainerArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	m.conM.RLock()
	cont, ok := m.containers[args.Container]
	m.conM.RUnlock()
	if !ok {
		return nil, fmt.Errorf("container does not exist")
	}

	job, err := pm.GetManager().System(
		"ip", "netns", "exec", fmt.Sprintf("%d", args.Container),
		"zerotier-cli", "-j", fmt.Sprintf("-D%s", cont.zerotierHome()), "info",
	)

	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal([]byte(job.Streams.Stdout()), &data); err != nil {
		return nil, err
	}

	return data, nil
}

func (m *containerManager) ztList(cmd *core.Command) (interface{}, error) {
	var args ContainerArguments
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	m.conM.RLock()
	cont, ok := m.containers[args.Container]
	m.conM.RUnlock()
	if !ok {
		return nil, fmt.Errorf("container does not exist")
	}

	job, err := pm.GetManager().System(
		"ip", "netns", "exec", fmt.Sprintf("%d", args.Container),
		"zerotier-cli", "-j", fmt.Sprintf("-D%s", cont.zerotierHome()), "listnetworks",
	)

	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal([]byte(job.Streams.Stdout()), &data); err != nil {
		return nil, err
	}

	return data, nil
}
