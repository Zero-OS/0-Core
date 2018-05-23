package builtin

import (
	"encoding/json"
	"fmt"

	"github.com/zero-os/0-core/base/pm"
)

type rtinfoData struct {
	job   pm.Job
	Disks []string `json:"disks"`
}
type rtinfoMgr struct {
	rtinfoMap map[string]*rtinfoParams
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

func rtinfoMkName(host string, port uint) string {
	return fmt.Sprintf("rtinfoclient-%s-%d", host, port)
}

func (rtm *rtinfoMgr) start(cmd *pm.Command) (interface{}, error) {
	var args rtinfoParams
	cmdbin := "rtinfo-client"
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	var cmdargs []string
	cmdargs = append(cmdargs, "--host", args.Host, "--port", fmt.Sprintf("%d", args.Port))

	for _, d := range args.Disks {
		cmdargs = append(cmdargs, "--disk", d)
	}

	key := fmt.Sprintf("%s:%d", args.Host, args.Port)
	if _, exists := rtm.rtinfoMap[key]; exists {
		return nil, pm.BadRequestError(fmt.Errorf("rtinfo running already: %s", key))
	}

	rtm.rtinfoMap[key] = &args
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
			delete(rtm.rtinfoMap, key)
		},
	}

	log.Debugf("rtinfo: %s started", key)
	job, err := pm.Run(rtinfocmd, onExit)
	if err != nil {
		return nil, err
	}
	rtm.rtinfoMap[key].job = job.Command().ID

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
	rtinfoParams, exists := rtm.rtinfoMap[key]

	if !exists {
		return true, nil
	}

	if err := pm.Kill(rtinfoParams.job); err != nil {
		return nil, err
	}
	delete(rtm.rtinfoMap, key)
	return true, nil
}
