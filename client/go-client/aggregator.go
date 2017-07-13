package client

type AggregatorManager interface {
	Query() (interface{}, error)
}

func Aggregator(cl Client) AggregatorManager {
	return &AggregatorMgr{cl}
}

type AggregatorMgr struct {
	Client
}

func (b *AggregatorMgr) Query() (interface{}, error) {
	var stats interface{}

	res, err := sync(b, "aggregator.query", A{})
	if err != nil {
		return stats, err
	}

	err = res.Json(&stats)
	return stats, err
}
