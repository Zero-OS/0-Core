package client

import (
	"encoding/json"
	"fmt"
)

const (
	//StateSuccess successs exit status
	StateSuccess = State("SUCCESS")

	//StateError error exist status
	StateError = State("ERROR")

	//StateTimeout timeout exit status
	StateTimeout = State("TIMEOUT")

	//StateKilled killed exit status
	StateKilled = State("KILLED")

	//StateUnknownCmd unknown cmd exit status
	StateUnknownCmd = State("UNKNOWN_CMD")

	//StateDuplicateID dublicate id exit status
	StateDuplicateID = State("DUPILICATE_ID")

	LevelJson = 20
)

type State string

type Streams []string

func (s Streams) Stdout() string {
	if len(s) >= 1 {
		return s[0]
	}
	return ""
}

func (s Streams) Stderr() string {
	if len(s) >= 2 {
		return s[1]
	}
	return ""
}

func (s Streams) String() string {
	return fmt.Sprintf("STDOUT:\n%s\nSTDERR:\n%s\n", s.Stdout(), s.Stderr())
}

type Result struct {
	ID        string  `json:"id"`
	Command   string  `json:"command"`
	Data      string  `json:"data"`
	Streams   Streams `json:"streams,omitempty"`
	Critical  string  `json:"critical,omitempty"`
	Level     int     `json:"level"`
	State     State   `json:"state"`
	StartTime int64   `json:"starttime"`
	Time      int64   `json:"time"`
	Tags      string  `json:"tags"`
	Container uint64  `json:"container"`
}

func (r *Result) Json(v interface{}) error {
	if r.Level != LevelJson {
		return fmt.Errorf("invalid result level, expecting %d", LevelJson)
	}

	return json.Unmarshal([]byte(r.Data), v)
}
