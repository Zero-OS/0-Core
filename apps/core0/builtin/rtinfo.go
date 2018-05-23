package builtin

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pborman/uuid"
	"github.com/zero-os/0-core/base/pm"
)

const cmdbin = "rtinfo-client"

type rtinfoMgr struct {
	info map[string]*rtinfoParams
	m    sync.RWMutex
}

type rtinfoParams struct {
	Host  string   `json:"host"`
	Port  uint     `json:"port"`
	Disks []string `json:"disks"`
	job   string
}

func init() {
	rtm := &rtinfoMgr{info: make(map[string]*rtinfoParams)}
	pm.RegisterBuiltIn("rtinfo.start", rtm.start)
	pm.RegisterBuiltIn("rtinfo.list", rtm.list)
	pm.RegisterBuiltIn("rtinfo.stop", rtm.stop)
}

func (rtm *rtinfoMgr) start(cmd *pm.Command) (interface{}, error) {
	var args rtinfoParams
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	cmdargs := []string{
		"--host", args.Host, "--port", fmt.Sprintf("%d", args.Port),
	}

	for _, d := range args.Disks {
		cmdargs = append(cmdargs, "--disk", d)
	}

	rtm.m.Lock()
	defer rtm.m.Unlock()

	key := fmt.Sprintf("%s:%d", args.Host, args.Port)
	_, exists := rtm.info[key]
	if exists {
		return nil, pm.NotAcceptableError("an rtinfo agent running already for this daemon")
	}

	args.job = uuid.New()
	rtm.info[key] = &args

	rtinfocmd := &pm.Command{
		ID:      args.job,
		Command: pm.CommandSystem,
		Arguments: pm.MustArguments(
			pm.SystemCommandArguments{
				Name: cmdbin,
				Args: cmdargs,
			},
		),
	}

	onExit := &pm.ExitHook{
		Action: func(state bool) {
			rtm.m.Lock()
			delete(rtm.info, key)
			rtm.m.Unlock()
		},
	}
	log.Debugf("rtinfo: %s started", key)

	_, err := pm.Run(rtinfocmd, onExit)

	return nil, err
}

func (rtm *rtinfoMgr) list(cmd *pm.Command) (interface{}, error) {

	return rtm.info, nil
}

func (rtm *rtinfoMgr) stop(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Host string `json:"host"`
		Port uint   `json:"port"`
	}
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s:%d", args.Host, args.Port)
	rtm.m.RLock()
	defer rtm.m.RUnlock()
	info, exists := rtm.info[key]

	if !exists {
		return nil, nil
	}

	return nil, pm.Kill(info.job)
}
