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
	"github.com/zero-os/0-core/base/pm/stream"
	"github.com/zero-os/0-core/base/settings"
	"github.com/zero-os/0-core/base/utils"
)

const (
	AggreagteAverage    = "A"
	AggreagteDifference = "D"
)

var (
	MaxJobs           int
	UnknownCommandErr = errors.New("unkonw command")
	DuplicateIDErr    = errors.New("duplicate job id")
)

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

//PM is the main r manager.
var (
	log = logging.MustGetLogger("pm")

	n        sync.Once
	jobs     map[string]Job
	jobsM    sync.RWMutex
	jobsCond *sync.Cond

	//needs clean up
	handlers []Handler
	queue    Queue

	pids    map[int]chan syscall.WaitStatus
	pidsMux sync.Mutex

	unprivileged bool
)

//NewPM creates a new PM
func New() {
	n.Do(func() {
		log.Debugf("initializing r manager")
		jobs = make(map[string]Job)
		jobsCond = sync.NewCond(&sync.Mutex{})
		pids = make(map[int]chan syscall.WaitStatus)
	})
}

func AddHandle(handler Handler) {
	handlers = append(handlers, handler)
}

func SetUnprivileged() {
	unprivileged = true
}

func RunFactory(cmd *Command, factory ProcessFactory, hooks ...RunnerHook) (Job, error) {
	if len(cmd.ID) == 0 {
		cmd.ID = uuid.New()
	}

	for _, handler := range handlers {
		if handler, ok := handler.(PreHandler); ok {
			handler.Pre(cmd)
		}
	}

	jobsM.Lock()
	defer jobsM.Unlock()

	_, exists := jobs[cmd.ID]
	if exists {
		return nil, DuplicateIDErr
	}

	job := newJob(cmd, factory, hooks...)
	jobs[cmd.ID] = job

	queue.Push(job)
	return job, nil
}

//Run runs a command immediately (no pre-processors)
func Run(cmd *Command, hooks ...RunnerHook) (Job, error) {
	factory := GetProcessFactory(cmd)
	if factory == nil {
		return nil, UnknownCommandErr
	}

	return RunFactory(cmd, factory, hooks...)
}

func loop() {
	ch := queue.Start()
	for {
		jobsCond.L.Lock()

		for len(jobs) >= MaxJobs {
			jobsCond.Wait()
		}
		jobsCond.L.Unlock()
		job := <-ch
		log.Debugf("starting job: %s", job.Command())
		go job.start(unprivileged)
	}
}

func processWait() {
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

			//Avoid reading the r state before the PIDTable call is complete.
			pidsMux.Lock()
			ch, ok := pids[pid]
			pidsMux.Unlock()

			if ok {
				go func(ch chan syscall.WaitStatus, status syscall.WaitStatus) {
					ch <- status
					close(ch)
					pidsMux.Lock()
					defer pidsMux.Unlock()
					delete(pids, pid)
				}(ch, status)
			}
		}

	}
}

func registerPID(g GetPID) error {
	pidsMux.Lock()
	defer pidsMux.Unlock()
	pid, err := g()
	if err != nil {
		return err
	}

	ch := make(chan syscall.WaitStatus)
	pids[pid] = ch

	return nil
}

func waitPID(pid int) syscall.WaitStatus {
	pidsMux.Lock()
	c, ok := pids[pid]
	pidsMux.Unlock()
	if !ok {
		return syscall.WaitStatus(0)
	}
	return <-c
}

//Start starts the r manager.
func Start() {
	//r and start all commands according to args.
	go processWait()
	go loop()
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
func RunSlice(slice settings.StartupSlice) {
	var all []string
	for _, startup := range slice {
		all = append(all, startup.Key())
	}

	state := newStateMachine(all...)
	cmdline := utils.GetKernelOptions().GetLast()

	for _, startup := range slice {
		if startup.Args == nil {
			startup.Args = make(map[string]interface{})
		}

		processArgs(startup.Args, cmdline)

		cmd := &Command{
			ID:              startup.Key(),
			Command:         startup.Name,
			RecurringPeriod: startup.RecurringPeriod,
			MaxRestart:      startup.MaxRestart,
			Protected:       startup.Protected,
			Tags:            startup.Tags,
			Arguments:       MustArguments(startup.Args),
		}

		go func(up settings.Startup, c *Command) {
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
				//NOTE: If r match is provided it take presence over the delay
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

			_, err := Run(c, hooks...)
			if err != nil {
				//failed to dispatch command to r manager.
				log.Errorf("failed to start command %v: %s", c, err)
				state.Release(c.ID, false)
			}
		}(startup, cmd)
	}

	//wait for the full slice to run
	log.Infof("Waiting for the slice to boot")
	state.WaitAll()
}

func cleanUp(runner Job) {
	jobsM.Lock()
	delete(jobs, runner.Command().ID)
	jobsM.Unlock()

	queue.Notify(runner)
	jobsCond.Broadcast()
}

//Processes returs a list of running processes
func Jobs() map[string]Job {
	res := make(map[string]Job)
	jobsM.RLock()
	defer jobsM.RUnlock()

	for k, v := range jobs {
		res[k] = v
	}

	return res
}

func JobOf(id string) (Job, bool) {
	jobsM.RLock()
	defer jobsM.RUnlock()
	r, ok := jobs[id]
	return r, ok
}

//Killall kills all running processes.
func Killall() {
	jobsM.RLock()
	defer jobsM.RUnlock()

	for _, v := range jobs {
		if v.Command().Protected {
			continue
		}
		v.Signal(syscall.SIGTERM)
	}
}

//Kill kills a r by the cmd ID
func Kill(cmdID string) error {
	jobsM.RLock()
	defer jobsM.RUnlock()
	v, ok := jobs[cmdID]
	if !ok {
		return fmt.Errorf("not found")
	}
	v.Signal(syscall.SIGTERM)
	return nil
}

func Aggregate(op, key string, value float64, id string, tags ...Tag) {
	for _, handler := range handlers {
		if handler, ok := handler.(StatsHandler); ok {
			handler.Stats(op, key, value, id, tags...)
		}
	}
}

func handleStatsMessage(cmd *Command, msg *stream.Message) {
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
	Aggregate(optype, key, v, id, tags...)
}

func msgCallback(cmd *Command, msg *stream.Message) {
	//handle stats messages
	if msg.Meta.Assert(stream.LevelStatsd) {
		handleStatsMessage(cmd, msg)
	}

	//update message
	msg.Epoch = time.Now().UnixNano()
	if cmd.Stream {
		msg.Meta = msg.Meta.Set(stream.StreamFlag)
	}

	for _, handler := range handlers {
		if handler, ok := handler.(MessageHandler); ok {
			handler.Message(cmd, msg)
		}
	}
}

func callback(cmd *Command, result *JobResult) {
	result.Tags = cmd.Tags
	for _, handler := range handlers {
		if handler, ok := handler.(ResultHandler); ok {
			handler.Result(cmd, result)
		}
	}
}

//System is a wrapper around core.system
func System(bin string, args ...string) (*JobResult, error) {
	runner, err := Run(&Command{
		ID:      uuid.New(),
		Command: CommandSystem,
		Arguments: MustArguments(
			SystemCommandArguments{
				Name: bin,
				Args: args,
			},
		),
	})

	if err != nil {
		return nil, err
	}

	job := runner.Wait()
	if job.State != StateSuccess {
		return job, fmt.Errorf("exited with error (%s): %v", job.State, job.Streams)
	}

	return job, nil
}
