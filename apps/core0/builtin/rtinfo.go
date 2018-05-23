package builtin

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/zero-os/0-core/base/pm"
)

const cmdbin = "rtinfo-client"

type rtinfoData struct {
	job   pm.Job
	Disks []string `json:"disks"`
}
type rtinfoMgr struct {
	rtinfoMap      map[string]*rtinfoParams
	rtinfoMapMutex sync.RWMutex
}

type rtinfoParams struct {
	Host  string   `json:"host"`
	Port  uint     `json:"port"`
	Disks []string `json:"disks"`
	job   string
}

func init() {
	rtm := &rtinfoMgr{rtinfoMap: make(map[string]*rtinfoParams)}
	pm.RegisterBuiltIn("rtinfo.start", rtm.start)
	pm.RegisterBuiltIn("rtinfo.list", rtm.list)
	pm.RegisterBuiltIn("rtinfo.stop", rtm.stop)
}

func (rtm *rtinfoMgr) getRtinfoParams(key string) (*rtinfoParams, bool) {
	rtm.rtinfoMapMutex.RLock()
	defer rtm.rtinfoMapMutex.RUnlock()
	rtinfoParams, exists := rtm.rtinfoMap[key]
	return rtinfoParams, exists
}

func rtinfoMkName(host string, port uint) string {
	return fmt.Sprintf("rtinfoclient-%s-%d", host, port)
}

func (rtm *rtinfoMgr) start(cmd *pm.Command) (interface{}, error) {
	var args rtinfoParams
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	cmdargs := []string{"--host", args.Host, "--port", fmt.Sprintf("%d", args.Port)}

	for _, d := range args.Disks {
		cmdargs = append(cmdargs, "--disk", d)
	}

	key := fmt.Sprintf("%s:%d", args.Host, args.Port)
	_, exists := rtm.getRtinfoParams(key)
	if exists {
		return nil, pm.BadRequestError(fmt.Errorf("rtinfo running already: %s", key))
	}

	rtinfocmd := &pm.Command{
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
			rtm.rtinfoMapMutex.Lock()
			delete(rtm.rtinfoMap, key)
			rtm.rtinfoMapMutex.Unlock()
		},
	}
	log.Debugf("rtinfo: %s started", key)

	rtm.rtinfoMapMutex.Lock()
	rtm.rtinfoMap[key] = &args
	rtm.rtinfoMapMutex.Unlock()

	job, err := pm.Run(rtinfocmd, onExit)
	if err != nil {
		return nil, err
	}
	rtm.rtinfoMapMutex.Lock()
	rtm.rtinfoMap[key].job = job.Command().ID
	rtm.rtinfoMapMutex.Unlock()

	return nil, nil
}

func (rtm *rtinfoMgr) list(cmd *pm.Command) (interface{}, error) {

	return rtm.rtinfoMap, nil
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
	rtinfoParams, exists := rtm.getRtinfoParams(key)

	if !exists {
		return true, nil
	}

	if err := pm.Kill(rtinfoParams.job); err != nil {
		return nil, err
	}
	return true, nil
}
