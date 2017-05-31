package client

import (
	"fmt"
)

type A map[string]interface{}

type Command struct {
	ID              string `json:"id"`
	Command         string `json:"command"`
	Arguments       A      `json:"arguments"`
	Queue           string `json:"queue"`
	StatsInterval   int    `json:"stats_interval,omitempty"`
	MaxTime         int    `json:"max_time,omitempty"`
	MaxRestart      int    `json:"max_restart,omitempty"`
	RecurringPeriod int    `json:"recurring_period,omitempty"`
	LogLevels       []int  `json:"log_levels,omitempty"`
	Tags            string `json:"tags"`
}

type Option interface {
	apply(cmd *Command)
}

type JobId string
type ProcessId uint64

type Client interface {
	Raw(command string, args A, opts ...Option) (JobId, error)
	Result(job JobId, timeout ...int) (*Result, error)
	Exists(job JobId) bool
	ResultNonBlock(job JobId) (*Result, error)
}

func sync(c Client, command string, args A, opts ...Option) (*Result, error) {
	j, err := c.Raw(command, args, opts...)
	if err != nil {
		return nil, err
	}
	res, err := c.Result(j, ResultDefaultTimeout)
	if err != nil {
		return nil, err
	}

	if res.State != StateSuccess {
		return res, fmt.Errorf("result is in state (%s): %s - data(%s)", res.State, res.Streams, res.Data)
	}

	return res, nil
}
