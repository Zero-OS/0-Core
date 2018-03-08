package network

import "fmt"

type Protocol interface {
	Configure(mgr NetworkManager, inf string) error
}

var (
	protocols = map[string]Protocol{}
)

func GetProtocol(name string) (Protocol, error) {
	if proto, ok := protocols[name]; ok {
		return proto, nil
	}

	return nil, fmt.Errorf("unknown protocol '%s'", name)
}
