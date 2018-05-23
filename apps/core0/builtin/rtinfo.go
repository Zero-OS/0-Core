package builtin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/zero-os/0-core/base/pm"
)

type rtinfoData struct {
	job   pm.Job
	Disks []string `json:"disks"`
}
type rtinfoMgr struct {
	rtinfoMap map[string]rtinfoData
}

type rtinfoParams struct {
	Host  string   `json:"host"`
	Port  uint     `json:"port"`
	Disks []string `json:"disks"`
}

func init() {
	rtm := &rtinfoMgr{rtinfoMap: make(map[string]rtinfoData)}
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
	procName := rtinfoMkName(args.Host, args.Port)
	if _, exists := rtm.rtinfoMap[procName]; exists {
		return nil, pm.BadRequestError(fmt.Errorf("rtinfo running already: %s", procName))
	}

	rtinfocmd := &pm.Command{
		ID:      procName,
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
			if !state {
				log.Errorf("rtinfoclient %s exited with an error", procName)
			}
		},
	}

	log.Debugf("rtinfo: %s started", procName)
	job, err := pm.Run(rtinfocmd, onExit)
	if err != nil {
		return nil, err
	}
	rtm.rtinfoMap[procName] = rtinfoData{job: job, Disks: args.Disks}

	return nil, nil
}

func (rtm *rtinfoMgr) list(cmd *pm.Command) (interface{}, error) {
	clientInfos := make([]rtinfoParams, 0, len(rtm.rtinfoMap))

	for k, v := range rtm.rtinfoMap {
		ci := rtinfoParams{}
		parts := strings.Split(k, "-")
		host, strPort := parts[1], parts[2]
		ci.Host = host
		port, err := strconv.Atoi(strPort)
		if err != nil {
			return nil, err
		}
		ci.Port = uint(port)
		ci.Disks = v.Disks
		clientInfos = append(clientInfos, ci)
	}
	return clientInfos, nil
}

func (rtm *rtinfoMgr) stop(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Host string `json:"host"`
		Port uint   `json:"port"`
	}
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	procName := rtinfoMkName(args.Host, args.Port)
	if err := pm.Kill(procName); err != nil {
		return nil, err
	}

	return true, nil
}
