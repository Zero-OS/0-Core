package builtin

import (
	"encoding/json"
	"github.com/Jumpscale/agent2/agent/lib/pm"
	"github.com/shirou/gopsutil/cpu"
)

const (
	CmdGetCpuInfo = "get_cpu_info"
)

func init() {
	pm.CmdMap[CmdGetCpuInfo] = InternalProcessFactory(getCPUInfo)
}

func getCPUInfo(cmd *pm.Cmd, cfg pm.RunCfg) *pm.JobResult {
	result := pm.NewBasicJobResult(cmd)
	result.Level = pm.L_RESULT_JSON

	info, err := cpu.CPUInfo()

	if err != nil {
		result.State = pm.S_ERROR
		m, _ := json.Marshal(err)
		result.Data = string(m)
	} else {
		result.State = pm.S_SUCCESS
		m, _ := json.Marshal(info)

		result.Data = string(m)
	}

	return result
}
