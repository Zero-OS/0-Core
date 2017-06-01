package client

type ZerotierItem struct {
	Nwid              string   `json:"nwid"`
	Name              string   `json:"name"`
	Status            string   `json:"status"`
	Type              string   `json:"type"`
	AllowDefault      bool     `json:"allowDefault"`
	AllowGlobal       bool     `json:"allowGlobal"`
	AllowManaged      bool     `json:"allowManaged"`
	Bridge            bool     `json:"bridge"`
	BroadcastEnabled  bool     `json:"broadcastEnabled"`
	DHCP              bool     `json:"dhcp"`
	ID                string   `json:"id"`
	Mac               string   `json:"mac"`
	Mtu               uint64   `json:"mtu"`
	NetConfRevision   uint64   `json:"netconfRevision"`
	PortDeviceName    string   `json:"portDeviceName"`
	PortError         uint8    `json:"portError"`
	AssignedAddresses []string `json:"assignedAddresses"`
}

type ZerotierRoute struct {
	Flags  uint64 `json:"flags"`
	Metric uint64 `json:"metric"`
	Target string `json:"target"`
	Via    string `json:"via"`
}

type ZerotierManager interface {
	List() ([]ZerotierItem, error)
}

func Zerotier(cl Client) ZerotierManager {
	return &zerotierMgr{cl}
}

type zerotierMgr struct {
	Client
}

func (b *zerotierMgr) List() ([]ZerotierItem, error) {
	zerotiers := []ZerotierItem{}

	res, err := sync(b, "zerotier.list", A{})
	if err != nil {
		return zerotiers, err
	}

	err = res.Json(&zerotiers)
	return zerotiers, err
}
