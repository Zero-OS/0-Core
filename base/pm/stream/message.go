package stream

import (
	"fmt"
)

const (
	//Message Flags
	StreamFlag Flag = 1 << iota
	//EOP success
	ExitSuccessFlag
	//EOP error
	ExitErrorFlag

	//LevelStdout stdout message
	LevelStdout uint16 = 1 // stdout
	//LevelStderr stderr message
	LevelStderr uint16 = 2 // stderr
	//LevelPublic public message
	LevelPublic uint16 = 3 // message for endusers / public message
	//LevelOperator operator message
	LevelOperator uint16 = 4 // message for operator / internal message
	//LevelUnknown unknown message
	LevelUnknown uint16 = 5 // log msg (unstructured = level5, cat=unknown)
	//LevelStructured structured message
	LevelStructured uint16 = 6 // log msg structured
	//LevelWarning warning message
	LevelWarning uint16 = 7 // warning message
	//LevelOpsError ops error message
	LevelOpsError uint16 = 8 // ops error
	//LevelCritical critical message
	LevelCritical uint16 = 9 // critical error
	//LevelStatsd statsd message
	LevelStatsd uint16 = 10 // statsd message(s) AVG
	//LevelDebug debug message
	LevelDebug uint16 = 11 // debug message
	//LevelResultJSON json result message
	LevelResultJSON uint16 = 20 // result message, json
	//LevelResultYAML yaml result message
	LevelResultYAML uint16 = 21 // result message, yaml
	//LevelResultTOML toml result message
	LevelResultTOML uint16 = 22 // result message, toml
	//LevelResultHRD hrd result message
	LevelResultHRD uint16 = 23 // result message, hrd
	//LevelResultJob job result message
	LevelResultJob uint16 = 30 // job, json (full result of a job)
)

var (
	ResultMessageLevels = []uint16{LevelResultJSON,
		LevelResultYAML, LevelResultTOML, LevelResultHRD, LevelResultJob}
)

type Flag uint16

type Meta uint64

func NewMeta(level uint16, flag ...Flag) Meta {
	m := uint32(level) << 16
	for _, f := range flag {
		m |= uint32(f)
	}

	return Meta(m)
}

func NewMetaWithCode(code uint32, level uint16, flag ...Flag) Meta {
	meta := NewMeta(level, flag...)
	return (Meta(code) << 32) | meta
}

func (m Meta) Level() uint16 {
	return uint16(uint64(m) >> 16 & 0xffff)
}

func (m Meta) Assert(level ...uint16) bool {
	l := uint16(uint64(m) >> 16 & 0xffff)
	for _, lv := range level {
		if l == lv {
			return true
		}
	}

	return false
}

//Is checks if a flag is set on the meta object
func (m Meta) Is(flag Flag) bool {
	return (uint16(m) & uint16(flag)) != 0
}

//Set sets a flag on meta object
func (m Meta) Set(flag Flag) Meta {
	return Meta(uint64(m) | uint64(flag))
}

//Code exit code
func (m Meta) Code() uint32 {
	return uint32(m >> 32)
}

//Base gets meta without the code part (used for backward compatibility)
func (m Meta) Base() Meta {
	return m & 0xffffffff
}

//Message is a message from running process
type Message struct {
	Message string `json:"message"`
	Epoch   int64  `json:"epoch"`
	Meta    Meta   `json:"meta"`
}

//MessageHandler represents a callback type
type MessageHandler func(*Message)

//String represents a message as a string
func (msg *Message) String() string {
	return fmt.Sprintf("%d|%s", msg.Meta.Level(), msg.Message)
}
