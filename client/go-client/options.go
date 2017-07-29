package client

type maxTimeOpt struct {
	timeout int
}

func (o maxTimeOpt) apply(cmd *Command) {
	cmd.MaxTime = o.timeout
}

func MaxTime(timeout int) Option {
	return maxTimeOpt{timeout}
}

type queueOpt struct {
	queue string
}

func (o queueOpt) apply(cmd *Command) {
	cmd.Queue = o.queue
}

func Queue(queue string) Option {
	return queueOpt{queue}
}

type maxRestartOpt struct {
	restart int
}

func (o maxRestartOpt) apply(cmd *Command) {
	cmd.MaxRestart = o.restart
}

func MaxRestart(restart int) Option {
	return maxRestartOpt{restart}
}

type recurringPeriodOpt struct {
	period int
}

func (o recurringPeriodOpt) apply(cmd *Command) {
	cmd.RecurringPeriod = o.period
}

func RecurringPeriod(period int) Option {
	return recurringPeriodOpt{period}
}

type idOpt struct {
	id string
}

func (o idOpt) apply(cmd *Command) {
	cmd.ID = o.id
}

func ID(id string) Option {
	return idOpt{id}
}
