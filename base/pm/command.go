package pm

import (
	"encoding/json"
	"fmt"
)

//Tags defines a list of keyword tags
type Tags []string

//JobFlags to control job behavior but only from the internal API
//Clients can't set the JobFlags, unlike the other public flags on the Command struct body.
type JobFlags struct {
	Protected bool
	NoOutput  bool
	NoSetPGID bool //set new process group id for job
}

//Command is the main way to communicate witht he process manager
//A Command.command is matched against a list of know process factories
//that build the corresponding process to handle the rest of the command
//arguments.
type Command struct {
	//Unique ID of the command, sets the job id
	ID string `json:"id"`
	//Command is the command name
	Command string `json:"command"`
	//Arguments, handled by the process
	Arguments *json.RawMessage `json:"arguments"`
	//Queue if set, commands with same queue are run synchronusly
	Queue string `json:"queue"`
	//StatsInterval fine tune when process statistics should be collected
	StatsInterval int `json:"stats_interval,omitempty"`
	//MaxTime max running time of the process, or it will get terminated
	MaxTime int `json:"max_time,omitempty"`
	//MaxRestart how many times the process manager should restart this process, if it failes
	MaxRestart int `json:"max_restart,omitempty"`
	//RecurringPeriod for recurring commands, defines how long it should wait between each run
	RecurringPeriod int `json:"recurring_period,omitempty"`
	//Stream if set to true, real time output of the process will get streamed over the output
	//channel
	Stream bool `json:"stream"`
	//LogLevels sets which log levels are to be logged
	LogLevels []int `json:"log_levels,omitempty"`
	//Tags custom user tags to be attached to the job
	Tags Tags `json:"tags"`

	//For internal use only, flags that can be set from inside the internal API
	Flags JobFlags `json:"-"`
}

//M short hand for map[string]interface{}
type M map[string]interface{}

//MustArguments serialize an object to *json.RawMessage
func MustArguments(args interface{}) *json.RawMessage {
	bytes, err := json.Marshal(args)
	if err != nil {
		panic(err)
	}

	raw := json.RawMessage(bytes)
	return &raw
}

//String represents cmd as a string
func (cmd *Command) String() string {
	return fmt.Sprintf("(%s# %s)", cmd.ID, cmd.Command)
}

//LoadCmd loads cmd from json string.
func LoadCmd(str []byte) (*Command, error) {
	var cmd Command
	err := json.Unmarshal(str, &cmd)
	if err != nil {
		return nil, err
	}

	return &cmd, err
}
