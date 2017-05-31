package client

type DiskManager interface {
	List() (Disks, error)
}

type BlockDevice struct {
	Name       string        `json:"name"`
	Kname      string        `json:"kname"`
	MajMin     string        `json:"maj:min"`
	Fstype     string        `json:"fstype"`
	Mountpoint string        `json:"mountpoint"`
	Label      string        `json:"label"`
	UUID       string        `json:"uuid"`
	Parttype   string        `json:"parttype"`
	Partlabel  string        `json:"partlabel"`
	Partuuid   string        `json:"partuuid"`
	Partflags  string        `json:"partflags"`
	Ra         string        `json:"ra"`
	Ro         string        `json:"ro"`
	Rm         string        `json:"rm"`
	Hotplug    string        `json:"hotplug"`
	Model      string        `json:"model"`
	Serial     string        `json:"serial"`
	Size       string        `json:"size"`
	State      string        `json:"state"`
	Owner      string        `json:"owner"`
	Group      string        `json:"group"`
	Mode       string        `json:"mode"`
	Alignment  string        `json:"alignment"`
	MinIo      string        `json:"min-io"`
	OptIo      string        `json:"opt-io"`
	PhySec     string        `json:"phy-sec"`
	LogSec     string        `json:"log-sec"`
	Rota       string        `json:"rota"`
	Sched      string        `json:"sched"`
	RqSize     string        `json:"rq-size"`
	Type       string        `json:"type"`
	DiscAln    string        `json:"disc-aln"`
	DiscGran   string        `json:"disc-gran"`
	DiscMax    string        `json:"disc-max"`
	DiscZero   string        `json:"disc-zero"`
	Wsame      string        `json:"wsame"`
	Wwn        string        `json:"wwn"`
	Rand       string        `json:"rand"`
	Pkname     string        `json:"pkname"`
	Hctl       string        `json:"hctl"`
	Tran       string        `json:"tran"`
	Subsystems string        `json:"subsystems"`
	Rev        string        `json:"rev"`
	Vendor     string        `json:"vendor"`
	Children   []BlockDevice `json:"children"`
}

type Disks struct {
	BlockDevices []BlockDevice `json:"blockdevices"`
}

type diskMgr struct {
	Client
}

func Disk(cl Client) DiskManager {
	return &diskMgr{cl}
}

func (d *diskMgr) List() (Disks, error) {
	var disks Disks

	res, err := sync(d, "disk.list", A{})
	if err != nil {
		return disks, err
	}

	return disks, res.Json(&disks)
}
