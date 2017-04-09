package network

const (
	NoneProtocol = "none"
)

func init() {
	protocols[NoneProtocol] = &noneProtocol{}
}

type noneProtocol struct{}

func (n *noneProtocol) Configure(mgr NetworkManager, inf string) error {
	log.Debugf("Bringing '%s' interface up", inf)

	return DeviceUp(inf)
}
