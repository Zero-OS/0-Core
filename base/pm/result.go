package pm

const (
	//StateSuccess successs exit status
	StateSuccess JobState = "SUCCESS"
	//StateError error exist status
	StateError JobState = "ERROR"
	//StateTimeout timeout exit status
	StateTimeout JobState = "TIMEOUT"
	//StateKilled killed exit status
	StateKilled JobState = "KILLED"
	//StateUnknownCmd unknown cmd exit status
	StateUnknownCmd JobState = "UNKNOWN_CMD"
	//StateDuplicateID dublicate id exit status
	StateDuplicateID JobState = "DUPILICATE_ID"
)

type JobState string

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

//JobResult represents a result of a job
type JobResult struct {
	ID        string   `json:"id"`
	Command   string   `json:"command"`
	Data      string   `json:"data"`
	Streams   Streams  `json:"streams,omitempty"`
	Critical  string   `json:"critical,omitempty"`
	Level     uint16   `json:"level"`
	State     JobState `json:"state"`
	Code      int      `json:"code"`
	StartTime int64    `json:"starttime"`
	Time      int64    `json:"time"`
	Tags      Tags     `json:"tags"`
	Container uint64   `json:"container"`
}

//NewJobResult creates a new job result from command
func NewJobResult(cmd *Command) *JobResult {
	return &JobResult{
		ID:      cmd.ID,
		Command: cmd.Command,
		Tags:    cmd.Tags,
	}
}
