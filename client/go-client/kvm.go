package client

type VM struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
	UUID  string `json:"uuid"`
}

type VMInfo struct {
	Block   map[string]DiskStats `json:"block"`
	Network map[string]Network   `json:"net"`
	Vcpu    map[string]VCpu      `json:"vcpu"`
}

type DiskStats struct {
	RdBytes uint64 `json:"rdbytes"`
	RdTimes uint64 `json:"rdtimes"`
	WrBytes uint64 `json:"wrbytes"`
	WrTimes uint64 `json:"wrtimes"`
}

type Network struct {
	RxBytes uint64 `json:"rxbytes"`
	RxPkts  uint64 `json:"rxpkts"`
	TxBytes uint64 `json:"txbytes"`
	TxPkts  uint64 `json:"txpkts"`
}

type VCpu struct {
	State float64 `json:"state"`
	Time  float64 `json:"time"`
}

type KvmManager interface {
	List() ([]VM, error)
	InfoPs(uuid string) (VMInfo, error)
}

func Kvm(cl Client) KvmManager {
	return &kvmMgr{cl}
}

type kvmMgr struct {
	Client
}

func (b *kvmMgr) List() ([]VM, error) {
	vms := []VM{}

	res, err := sync(b, "kvm.list", A{})
	if err != nil {
		return vms, err
	}

	err = res.Json(&vms)
	return vms, err
}

func (b *kvmMgr) InfoPs(uuid string) (VMInfo, error) {
	vmInfo := VMInfo{}

	res, err := sync(b, "kvm.infops", A{
		"uuid": uuid,
	})
	if err != nil {
		return vmInfo, err
	}

	err = res.Json(&vmInfo)
	return vmInfo, err
}
