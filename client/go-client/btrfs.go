package client

type BtrfsFS struct {
	Label        string        `json:"label"`
	UUID         string        `json:"uuid"`
	TotalDevices int           `json:"total_devices"`
	Used         int64         `json:"used"`
	Devices      []btrfsDevice `json:"devices"`
}

type btrfsDevice struct {
	Missing bool   `json:"missing,omitempty"`
	DevID   int    `json:"dev_id"`
	Size    int64  `json:"size"`
	Used    int64  `json:"used"`
	Path    string `json:"path"`
}

type BtrfsManager interface {
	List() ([]BtrfsFS, error)
}

func Btrfs(cl Client) BtrfsManager {
	return &btrfsMgr{cl}
}

type btrfsMgr struct {
	Client
}

func (b *btrfsMgr) List() ([]BtrfsFS, error) {
	fss := []BtrfsFS{}

	res, err := sync(b, "btrfs.list", A{})
	if err != nil {
		return fss, err
	}

	err = res.Json(&fss)
	return fss, err
}
