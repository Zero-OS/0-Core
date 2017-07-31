package pm

import "github.com/zero-os/0-core/base/pm/stream"

type Handler interface{}

type ResultHandler interface {
	Result(cmd *Command, result *JobResult)
}

type MessageHandler interface {
	Message(*Command, *stream.Message)
}

type StatsHandler interface {
	Stats(operation string, key string, value float64, id string, tags ...Tag)
}

type PreHandler interface {
	Pre(cmd *Command)
}
