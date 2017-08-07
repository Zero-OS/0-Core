package builtin

import (
	"encoding/json"
	"github.com/zero-os/0-core/base/pm"
	"os"
	"runtime"
	"runtime/pprof"
)

func init() {
	pm.RegisterBuiltIn("pprof.cpu.start", pprofStart)
	pm.RegisterBuiltIn("pprof.cpu.stop", pprofStop)
	pm.RegisterBuiltIn("pprof.mem.write", pprofMemWrite)
	pm.RegisterBuiltIn("pprof.mem.stat", pprofMemStat)
}

func pprofStart(cmd *pm.Command) (interface{}, error) {
	var args struct {
		File string `json:"file"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	fd, err := os.Create(args.File)
	if err != nil {
		return nil, err
	}

	return nil, pprof.StartCPUProfile(fd)
}

func pprofStop(cmd *pm.Command) (interface{}, error) {
	pprof.StopCPUProfile()
	return nil, nil
}

func pprofMemWrite(cmd *pm.Command) (interface{}, error) {
	var args struct {
		File string `json:"file"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	fd, err := os.Create(args.File)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return nil, pprof.WriteHeapProfile(fd)
}

func pprofMemStat(cmd *pm.Command) (interface{}, error) {
	var stat runtime.MemStats
	runtime.ReadMemStats(&stat)
	return stat, nil
}
