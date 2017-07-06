package core

import (
	"encoding/json"
	"fmt"
)

type Route string

//Cmd is an executable command
type Command struct {
	ID              string           `json:"id"`
	Command         string           `json:"command"`
	Arguments       *json.RawMessage `json:"arguments"`
	Queue           string           `json:"queue"`
	StatsInterval   int              `json:"stats_interval,omitempty"`
	MaxTime         int              `json:"max_time,omitempty"`
	MaxRestart      int              `json:"max_restart,omitempty"`
	RecurringPeriod int              `json:"recurring_period,omitempty"`
	Stream          bool             `json:"stream"`
	LogLevels       []int            `json:"log_levels,omitempty"`
	Tags            string           `json:"tags"`

	Route     Route `json:"-"`
	Protected bool  `json:"-"`
}

type M map[string]interface{}

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
