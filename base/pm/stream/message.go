package stream

import (
	"fmt"
	"github.com/zero-os/0-core/base/pm/core"
)

const (
	LevelInvalid = -1
	//LevelStdout stdout message
	LevelStdout = 1 // stdout
	//LevelStderr stderr message
	LevelStderr = 2 // stderr
	//LevelPublic public message
	LevelPublic = 3 // message for endusers / public message
	//LevelOperator operator message
	LevelOperator = 4 // message for operator / internal message
	//LevelUnknown unknown message
	LevelUnknown = 5 // log msg (unstructured = level5, cat=unknown)
	//LevelStructured structured message
	LevelStructured = 6 // log msg structured
	//LevelWarning warning message
	LevelWarning = 7 // warning message
	//LevelOpsError ops error message
	LevelOpsError = 8 // ops error
	//LevelCritical critical message
	LevelCritical = 9 // critical error
	//LevelStatsd statsd message
	LevelStatsd = 10 // statsd message(s) AVG
	//LevelDebug debug message
	LevelDebug = 11 // debug message
	//LevelResultJSON json result message
	LevelResultJSON = 20 // result message, json
	//LevelResultYAML yaml result message
	LevelResultYAML = 21 // result message, yaml
	//LevelResultTOML toml result message
	LevelResultTOML = 22 // result message, toml
	//LevelResultHRD hrd result message
	LevelResultHRD = 23 // result message, hrd
	//LevelResultJob job result message
	LevelResultJob = 30 // job, json (full result of a job)

	//Exit message (this message must be sent by all processes as a last message)
	//other wise the PM will assume ERROR exit status.
	LevelExitState = 50

	//LevelInternal specify the start of the internal log levels
	LevelInternal = 100

	//LevelInternalMonitorPid instruct the agent to consider the cpu and mem consumption
	//of that PID (in the message body)
	LevelInternalMonitorPid = 101
)

var (
	ResultMessageLevels = []int{LevelResultJSON,
		LevelResultYAML, LevelResultTOML, LevelResultHRD, LevelResultJob}

	MessageExitSuccess = &Message{
		Level:   LevelExitState,
		Message: core.StateSuccess,
	}

	MessageExitError = &Message{
		Level:   LevelExitState,
		Message: core.StateError,
	}
)

//Message is a message from running process
type Message struct {
	Level   int    `json:"level"`
	Message string `json:"message"`
	Epoch   int64  `json:"epoch"`
}

//MessageHandler represents a callback type
type MessageHandler func(*Message)

//String represents a message as a string
func (msg *Message) String() string {
	return fmt.Sprintf("%d|%s", msg.Level, msg.Message)
}
