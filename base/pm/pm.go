package pm

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/op/go-logging"
	"github.com/pborman/uuid"
	psutil "github.com/shirou/gopsutil/process"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"github.com/zero-os/0-core/base/pm/stream"
	"github.com/zero-os/0-core/base/settings"
	"github.com/zero-os/0-core/base/utils"
)

const (
	AggreagteAverage    = "A"
	AggreagteDifference = "D"
)

var (
	log               = logging.MustGetLogger("pm")
	UnknownCommandErr = errors.New("unkonw command")
	DuplicateIDErr    = errors.New("duplicate job id")
)

type PreProcessor func(cmd *core.Command)

//MeterHandler represents a callback type
type MeterHandler func(cmd *core.Command, p *psutil.Process)

type MessageHandler func(*core.Command, *stream.Message)

//ResultHandler represents a callback type
type ResultHandler func(cmd *core.Command, result *core.JobResult)

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

//StatsFlushHandler represents a callback type
type StatsHandler func(operation string, key string, value float64, id string, tags ...Tag)

//PM is the main process manager.
type PM struct {
	cmds    chan *core.Command
	runners map[string]Runner

	runnersMux sync.RWMutex
	maxJobs    int
	jobsCond   *sync.Cond

	preProcessors      []PreProcessor
	msgHandlers        []MessageHandler
	resultHandlers     []ResultHandler
	statsFlushHandlers []StatsHandler
	queueMgr           *cmdQueueManager

	pids    map[int]chan syscall.WaitStatus
	pidsMux sync.Mutex

	unprivileged bool
}

var pm *PM

//NewPM creates a new PM
func InitProcessManager(maxJobs int) *PM {
	pm = &PM{
		cmds:     make(chan *core.Command),
		runners:  make(map[string]Runner),
		maxJobs:  maxJobs,
		jobsCond: sync.NewCond(&sync.Mutex{}),
		queueMgr: newCmdQueueManager(),

		pids: make(map[int]chan syscall.WaitStatus),
	}

	log.Infof("Process manager intialization completed")
	return pm
}

//TODO: That's not clean, find another way to make this available for other
//code
func GetManager() *PM {
	if pm == nil {
		panic("Process manager is not intialized")
	}
	return pm
}

//PushCmd schedules a command to run, might block if no free slots available
//it also runs all the preprocessors
func (pm *PM) PushCmd(cmd *core.Command) {
	for _, processor := range pm.preProcessors {
		processor(cmd)
	}

	if cmd.Queue == "" {
		pm.cmds <- cmd
	} else {
		pm.pushCmdToQueue(cmd)
	}
}

func (pm *PM) pushCmdToQueue(cmd *core.Command) {
	pm.queueMgr.Push(cmd)
}

func (pm *PM) AddPreProcessor(processor PreProcessor) {
	pm.preProcessors = append(pm.preProcessors, processor)
}

//AddMessageHandler adds handlers for messages that are captured from sub processes. Logger can use this to
//process messages
func (pm *PM) AddMessageHandler(handler MessageHandler) {
	pm.msgHandlers = append(pm.msgHandlers, handler)
}

//AddResultHandler adds a handler that receives job results.
func (pm *PM) AddResultHandler(handler ResultHandler) {
	pm.resultHandlers = append(pm.resultHandlers, handler)
}

//AddStatsFlushHandler adds handler to stats flush.
func (pm *PM) AddStatsHandler(handler StatsHandler) {
	pm.statsFlushHandlers = append(pm.statsFlushHandlers, handler)
}

func (pm *PM) SetUnprivileged() {
	pm.unprivileged = true
}

func (pm *PM) NewRunner(cmd *core.Command, factory process.ProcessFactory, hooks ...RunnerHook) (Runner, error) {
	pm.runnersMux.Lock()
	defer pm.runnersMux.Unlock()

	_, exists := pm.runners[cmd.ID]
	if exists {
		return nil, DuplicateIDErr
	}

	runner := NewRunner(pm, cmd, factory, hooks...)
	pm.runners[cmd.ID] = runner

	go runner.start(pm.unprivileged)

	return runner, nil
}

//RunCmd runs a command immediately (no pre-processors)
func (pm *PM) RunCmd(cmd *core.Command, hooks ...RunnerHook) (runner Runner, err error) {
	factory := GetProcessFactory(cmd)
	defer func() {
		if err != nil {
			pm.queueMgr.Notify(cmd)
			pm.jobsCond.Broadcast()
		}
	}()

	if factory == nil {
		log.Errorf("Unknow command '%s'", cmd.Command)
		errResult := core.NewBasicJobResult(cmd)
		errResult.State = core.StateUnknownCmd
		pm.resultCallback(cmd, errResult)
		err = UnknownCommandErr
		return
	}

	runner, err = pm.NewRunner(cmd, factory, hooks...)

	if err == DuplicateIDErr {
		log.Errorf("Duplicate job id '%s'", cmd.ID)
		errResult := core.NewBasicJobResult(cmd)
		errResult.State = core.StateDuplicateID
		errResult.Data = err.Error()
		pm.resultCallback(cmd, errResult)
		return
	} else if err != nil {
		errResult := core.NewBasicJobResult(cmd)
		errResult.State = core.StateError
		errResult.Data = err.Error()
		pm.resultCallback(cmd, errResult)
		return
	}

	return
}

func (pm *PM) processCmds() {
	for {
		pm.jobsCond.L.Lock()

		for len(pm.runners) >= pm.maxJobs {
			pm.jobsCond.Wait()
		}
		pm.jobsCond.L.Unlock()

		var cmd *core.Command

		//we have 2 possible sources of cmds.
		//1- cmds that doesn't require waiting on a queue, those can run immediately
		//2- cmds that were waiting on a queue (so they must execute serially)
		select {
		case cmd = <-pm.cmds:
		case cmd = <-pm.queueMgr.Producer():
		}

		pm.RunCmd(cmd)
	}
}

func (pm *PM) processWait() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGCHLD)
	for range c {
		//we wait for sigchld
		for {
			//once we get a signal, we consume ALL the died children
			//since signal.Notify will not wait on channel writes
			//we create a buffer of 2 and on each signal we loop until wait gives an error
			var status syscall.WaitStatus
			var rusage syscall.Rusage

			log.Debug("Waiting for children")
			pid, err := syscall.Wait4(-1, &status, 0, &rusage)
			if err != nil {
				log.Debugf("wait error: %s", err)
				break
			}

			//Avoid reading the process state before the Register call is complete.
			pm.pidsMux.Lock()
			ch, ok := pm.pids[pid]
			pm.pidsMux.Unlock()

			if ok {
				go func(ch chan syscall.WaitStatus, status syscall.WaitStatus) {
					ch <- status
					close(ch)
					pm.pidsMux.Lock()
					defer pm.pidsMux.Unlock()
					delete(pm.pids, pid)
				}(ch, status)
			}
		}

	}
}

func (pm *PM) Register(g process.GetPID) error {
	pm.pidsMux.Lock()
	defer pm.pidsMux.Unlock()
	pid, err := g()
	if err != nil {
		return err
	}

	ch := make(chan syscall.WaitStatus)
	pm.pids[pid] = ch

	return nil
}

func (pm *PM) WaitPID(pid int) syscall.WaitStatus {
	pm.pidsMux.Lock()
	c, ok := pm.pids[pid]
	pm.pidsMux.Unlock()
	if !ok {
		return syscall.WaitStatus(0)
	}
	return <-c
}

//Run starts the process manager.
func (pm *PM) Run() {
	//process and start all commands according to args.
	go pm.processWait()
	go pm.processCmds()
}

func processArgs(args map[string]interface{}, values map[string]interface{}) {
	for key, value := range args {
		switch value := value.(type) {
		case string:
			args[key] = utils.Format(value, values)
		case []string:
			newstrlist := make([]string, len(value))
			for _, strvalue := range value {
				newstrlist = append(newstrlist, utils.Format(strvalue, values))
			}
			args[key] = newstrlist
		}
	}

}

/*
RunSlice runs a slice of processes honoring dependencies. It won't just
start in order, but will also make sure a service won't start until it's dependencies are
running.
*/
func (pm *PM) RunSlice(slice settings.StartupSlice) {
	var all []string
	for _, startup := range slice {
		all = append(all, startup.Key())
	}

	state := NewStateMachine(all...)
	cmdline := utils.GetKernelOptions().GetLast()

	for _, startup := range slice {
		if startup.Args == nil {
			startup.Args = make(map[string]interface{})
		}

		processArgs(startup.Args, cmdline)

		cmd := &core.Command{
			ID:              startup.Key(),
			Command:         startup.Name,
			RecurringPeriod: startup.RecurringPeriod,
			MaxRestart:      startup.MaxRestart,
			Protected:       startup.Protected,
			Tags:            startup.Tags,
			Arguments:       core.MustArguments(startup.Args),
		}

		go func(up settings.Startup, c *core.Command) {
			log.Debugf("Waiting for %s to run %s", up.After, cmd)
			canRun := state.Wait(up.After...)

			if !canRun {
				log.Errorf("Can't start %s because one of the dependencies failed", c)
				state.Release(c.ID, false)
				return
			}

			log.Infof("Starting %s", c)
			var hooks []RunnerHook

			if up.RunningMatch != "" {
				//NOTE: If runner match is provided it take presence over the delay
				hooks = append(hooks, &MatchHook{
					Match: up.RunningMatch,
					Action: func(msg *stream.Message) {
						log.Infof("Got '%s' from '%s' signal running", msg.Message, c.ID)
						state.Release(c.ID, true)
					},
				})
			} else if up.RunningDelay >= 0 {
				d := 2 * time.Second
				if up.RunningDelay > 0 {
					d = time.Duration(up.RunningDelay) * time.Second
				}

				hook := &DelayHook{
					Delay: d,
					Action: func() {
						state.Release(c.ID, true)
					},
				}
				hooks = append(hooks, hook)
			}

			hooks = append(hooks, &ExitHook{
				Action: func(s bool) {
					state.Release(c.ID, s)
				},
			})

			_, err := pm.RunCmd(c, hooks...)
			if err != nil {
				//failed to dispatch command to process manager.
				state.Release(c.ID, false)
			}
		}(startup, cmd)
	}

	//wait for the full slice to run
	log.Infof("Waiting for the slice to boot")
	state.WaitAll()
}

func (pm *PM) cleanUp(runner Runner) {
	pm.runnersMux.Lock()
	delete(pm.runners, runner.Command().ID)
	pm.runnersMux.Unlock()

	pm.queueMgr.Notify(runner.Command())
	pm.jobsCond.Broadcast()
}

//Processes returs a list of running processes
func (pm *PM) Runners() map[string]Runner {
	res := make(map[string]Runner)
	pm.runnersMux.RLock()
	defer pm.runnersMux.RUnlock()

	for k, v := range pm.runners {
		res[k] = v
	}

	return res
}

func (pm *PM) Runner(id string) (Runner, bool) {
	pm.runnersMux.RLock()
	defer pm.runnersMux.RUnlock()
	r, ok := pm.runners[id]
	return r, ok
}

//Killall kills all running processes.
func (pm *PM) Killall() {
	pm.runnersMux.RLock()
	defer pm.runnersMux.RUnlock()

	for _, v := range pm.runners {
		if v.Command().Protected {
			continue
		}
		v.Terminate()
	}
}

//Kill kills a process by the cmd ID
func (pm *PM) Kill(cmdID string) error {
	pm.runnersMux.RLock()
	defer pm.runnersMux.RUnlock()
	v, ok := pm.runners[cmdID]
	if !ok {
		return fmt.Errorf("not found")
	}
	v.Terminate()
	return nil
}

func (pm *PM) Aggregate(op, key string, value float64, id string, tags ...Tag) {
	for _, handler := range pm.statsFlushHandlers {
		handler(op, key, value, id, tags...)
	}
}

func (pm *PM) handleStatsMessage(cmd *core.Command, msg *stream.Message) {
	parts := strings.Split(msg.Message, "|")
	if len(parts) < 2 {
		log.Errorf("Invalid statsd string, expecting data|type[|options], got '%s'", msg.Message)
	}

	optype := parts[1]

	var tagsStr string
	if len(parts) == 3 {
		tagsStr = parts[2]
	}

	data := strings.Split(parts[0], ":")
	if len(data) != 2 {
		log.Errorf("Invalid statsd data, expecting key:value, got '%s'", parts[0])
	}

	key := strings.Trim(data[0], " ")
	value := data[1]
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Warning("invalid stats message value is not a number '%s'", msg.Message)
		return
	}

	parse := func(t string) (string, []Tag) {
		var tags []Tag
		var id string
		for _, p := range strings.Split(t, ",") {
			kv := strings.SplitN(p, "=", 2)
			var v string
			if len(kv) == 2 {
				v = kv[1]
			}
			//special tag id.
			if kv[0] == "id" {
				id = v
				continue
			}
			tags = append(tags, Tag{
				Key:   kv[0],
				Value: v,
			})
		}

		return id, tags
	}

	id, tags := parse(tagsStr)
	pm.Aggregate(optype, key, v, id, tags...)
}

func (pm *PM) msgCallback(cmd *core.Command, msg *stream.Message) {
	//handle stats messages
	if msg.Meta.Assert(stream.LevelStatsd) {
		pm.handleStatsMessage(cmd, msg)
	}

	//update message
	msg.Epoch = time.Now().UnixNano()
	if cmd.Stream {
		msg.Meta = msg.Meta.Set(stream.StreamFlag)
	}
	for _, handler := range pm.msgHandlers {
		handler(cmd, msg)
	}
}

func (pm *PM) resultCallback(cmd *core.Command, result *core.JobResult) {
	result.Tags = cmd.Tags
	//NOTE: we always force the real gid and nid on the result.

	for _, handler := range pm.resultHandlers {
		handler(cmd, result)
	}
}

//System is a wrapper around core.system
func (pm *PM) System(bin string, args ...string) (*core.JobResult, error) {
	runner, err := pm.RunCmd(&core.Command{
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: bin,
				Args: args,
			},
		),
	})

	if err != nil {
		return nil, err
	}

	job := runner.Wait()
	if job.State != core.StateSuccess {
		return job, fmt.Errorf("exited with error (%s): %v", job.State, job.Streams)
	}

	return job, nil
}
