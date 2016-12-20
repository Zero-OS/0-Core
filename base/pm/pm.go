package pm

import (
	"errors"
	"fmt"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/g8os/core0/base/pm/stream"
	"github.com/g8os/core0/base/settings"
	"github.com/g8os/core0/base/stats"
	"github.com/g8os/core0/base/utils"
	"github.com/op/go-logging"
	psutil "github.com/shirou/gopsutil/process"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	log               = logging.MustGetLogger("pm")
	UnknownCommandErr = errors.New("unkonw command")
	DuplicateIDErr    = errors.New("duplicate job id")
)

//MeterHandler represents a callback type
type MeterHandler func(cmd *core.Command, p *psutil.Process)

type MessageHandler func(*core.Command, *stream.Message)

//ResultHandler represents a callback type
type ResultHandler func(cmd *core.Command, result *core.JobResult)

//StatsFlushHandler represents a callback type
type StatsHandler func(operation stats.Operation, key string, value float64, tags string)

//PM is the main process manager.
type PM struct {
	midMux  sync.Mutex
	cmds    chan *core.Command
	runners map[string]Runner

	runnersMux sync.Mutex
	maxJobs    int
	jobsCond   *sync.Cond

	msgHandlers         []MessageHandler
	resultHandlers      []ResultHandler
	routeResultHandlers map[core.Route][]ResultHandler
	statsFlushHandlers  []StatsHandler
	queueMgr            *cmdQueueManager

	pids    map[int]chan *syscall.WaitStatus
	pidsMux sync.Mutex
}

var pm *PM

//NewPM creates a new PM
func InitProcessManager(maxJobs int) *PM {
	pm = &PM{
		cmds:                make(chan *core.Command),
		runners:             make(map[string]Runner),
		maxJobs:             maxJobs,
		jobsCond:            sync.NewCond(&sync.Mutex{}),
		routeResultHandlers: make(map[core.Route][]ResultHandler),
		queueMgr:            newCmdQueueManager(),

		pids: make(map[int]chan *syscall.WaitStatus),
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

func loadMid(midfile string) uint32 {
	content, err := ioutil.ReadFile(midfile)
	if err != nil {
		log.Errorf("%s", err)
		return 0
	}
	v, err := strconv.ParseUint(string(content), 10, 32)
	if err != nil {
		log.Errorf("%s", err)
		return 0
	}
	return uint32(v)
}

func saveMid(midfile string, mid uint32) {
	ioutil.WriteFile(midfile, []byte(fmt.Sprintf("%d", mid)), 0644)
}

//RunCmd runs and manage command
func (pm *PM) PushCmd(cmd *core.Command) {
	pm.cmds <- cmd
}

/*
RunCmdQueued Same as RunCmdAsync put will queue the command for later execution when there are no
other commands runs on the same queue.

The queue name is retrieved from cmd.Args[queue]
*/
func (pm *PM) PushCmdToQueue(cmd *core.Command) {
	pm.queueMgr.Push(cmd)
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

func (pm *PM) AddRouteResultHandler(route core.Route, handler ResultHandler) {
	pm.routeResultHandlers[route] = append(pm.routeResultHandlers[route], handler)
}

//AddStatsFlushHandler adds handler to stats flush.
func (pm *PM) AddStatsHandler(handler StatsHandler) {
	pm.statsFlushHandlers = append(pm.statsFlushHandlers, handler)
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

	go runner.Run()

	return runner, nil
}

func (pm *PM) RunCmd(cmd *core.Command, hooks ...RunnerHook) (Runner, error) {
	factory := GetProcessFactory(cmd)
	if factory == nil {
		log.Errorf("Unknow command '%s'", cmd.Command)
		errResult := core.NewBasicJobResult(cmd)
		errResult.State = core.StateUnknownCmd
		pm.resultCallback(cmd, errResult)
		return nil, UnknownCommandErr
	}

	runner, err := pm.NewRunner(cmd, factory, hooks...)

	if err == DuplicateIDErr {
		log.Errorf("Duplicate job id '%s'", cmd.ID)
		errResult := core.NewBasicJobResult(cmd)
		errResult.State = core.StateDuplicateID
		errResult.Data = err.Error()
		pm.resultCallback(cmd, errResult)
		return nil, err
	} else if err != nil {
		errResult := core.NewBasicJobResult(cmd)
		errResult.State = core.StateError
		errResult.Data = err.Error()
		pm.resultCallback(cmd, errResult)
		return nil, err
	}

	return runner, nil
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
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGCHLD)
	for _ = range ch {
		var status syscall.WaitStatus
		var rusage syscall.Rusage

		pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, &rusage)
		if err != nil {
			log.Errorf("Wait error: %s", err)
			continue
		}

		//Avoid reading the process state before the Register call is complete.
		pm.pidsMux.Lock()
		ch, ok := pm.pids[pid]
		pm.pidsMux.Unlock()

		if ok {
			go func() {
				ch <- &status
				close(ch)
				pm.pidsMux.Lock()
				defer pm.pidsMux.Unlock()
				delete(pm.pids, pid)
			}()
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

	ch := make(chan *syscall.WaitStatus)
	pm.pids[pid] = ch

	return nil
}

func (pm *PM) WaitPID(pid int) *syscall.WaitStatus {
	return <-pm.pids[pid]
}

//Run starts the process manager.
func (pm *PM) Run() {
	//process and start all commands according to args.
	go pm.processWait()
	go pm.processCmds()
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

	for _, startup := range slice {
		if startup.Args == nil {
			startup.Args = make(map[string]interface{})
		}

		cmd := &core.Command{
			ID:        startup.Key(),
			Command:   startup.Name,
			Arguments: core.MustArguments(startup.Args),
		}

		go func(up settings.Startup, c *core.Command) {
			log.Debugf("Waiting for %s to run %s", up.After, cmd)
			canRun := state.Wait(up.After...)

			if canRun {
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

				pm.RunCmd(c, hooks...)

			} else {
				log.Errorf("Can't start %s because one of the dependencies failed", c)
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
	return pm.runners
}

//Killall kills all running processes.
func (pm *PM) Killall() {
	pm.runnersMux.Lock()
	defer pm.runnersMux.Unlock()

	for _, v := range pm.runners {
		v.Kill()
	}
}

//Kill kills a process by the cmd ID
func (pm *PM) Kill(cmdID string) {
	v, o := pm.runners[cmdID]
	if o {
		v.Kill()
	}
}

func (pm *PM) handleStatsMessage(cmd *core.Command, msg *stream.Message) {
	parts := strings.Split(msg.Message, "|")
	if len(parts) < 2 {
		log.Errorf("Invalid statsd string, expecting data|type[|options], got '%s'", msg.Message)
	}

	optype := parts[1]

	var tags string
	if len(parts) == 3 {
		tags = parts[2]
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

	for _, handler := range pm.statsFlushHandlers {
		handler(stats.Operation(optype), key, v, tags)
	}
}

func (pm *PM) msgCallback(cmd *core.Command, msg *stream.Message) {
	//handle stats messages
	if msg.Level == stream.LevelStatsd {
		pm.handleStatsMessage(cmd, msg)
		return
	}

	//handle logging.
	if len(cmd.LogLevels) > 0 && !utils.In(cmd.LogLevels, msg.Level) {
		return
	}

	//stamp msg.
	msg.Epoch = time.Now().UnixNano()
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

	for _, handler := range pm.routeResultHandlers[cmd.Route] {
		handler(cmd, result)
	}
}
