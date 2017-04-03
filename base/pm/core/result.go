package core

const (
	//StateSuccess successs exit status
	StateSuccess = "SUCCESS"
	//StateError error exist status
	StateError = "ERROR"
	//StateTimeout timeout exit status
	StateTimeout = "TIMEOUT"
	//StateKilled killed exit status
	StateKilled = "KILLED"
	//StateUnknownCmd unknown cmd exit status
	StateUnknownCmd = "UNKNOWN_CMD"
	//StateDuplicateID dublicate id exit status
	StateDuplicateID = "DUPILICATE_ID"
)

//JobResult represents a result of a job
type JobResult struct {
	ID        string   `json:"id"`
	Command   string   `json:"command"`
	Data      string   `json:"data"`
	Streams   []string `json:"streams,omitempty"`
	Critical  string   `json:"critical,omitempty"`
	Level     int      `json:"level"`
	State     string   `json:"state"`
	StartTime int64    `json:"starttime"`
	Time      int64    `json:"time"`
	Tags      string   `json:"tags"`
	Container uint64   `json:"container"`
}

//NewBasicJobResult creates a new job result from command
func NewBasicJobResult(cmd *Command) *JobResult {
	return &JobResult{
		ID:      cmd.ID,
		Command: cmd.Command,
		Tags:    cmd.Tags,
	}
}
