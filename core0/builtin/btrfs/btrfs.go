package btrfs

import (
	"encoding/json"
	"fmt"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
	"regexp"
	"strconv"
	"strings"
)

var (
	reBtrfsFilesystemDf = regexp.MustCompile(`(?m:(\w+),\s(\w+):\s+total=(\d+),\s+used=(\d+))`)
)

type btrfsManager struct{}

func init() {
	var m btrfsManager

	pm.CmdMap["btrfs.list"] = process.NewInternalProcessFactory(m.List)
	pm.CmdMap["btrfs.info"] = process.NewInternalProcessFactory(m.Info)
	pm.CmdMap["btrfs.create"] = process.NewInternalProcessFactory(m.Create)
	pm.CmdMap["btrfs.device_add"] = process.NewInternalProcessFactory(m.DeviceAdd)
	pm.CmdMap["btrfs.device_remove"] = process.NewInternalProcessFactory(m.DeviceRemove)
	pm.CmdMap["btrfs.subvol_create"] = process.NewInternalProcessFactory(m.SubvolCreate)
	pm.CmdMap["btrfs.subvol_delete"] = process.NewInternalProcessFactory(m.SubvolDelete)
	pm.CmdMap["btrfs.subvol_quota"] = process.NewInternalProcessFactory(m.SubvolQuota)
	pm.CmdMap["btrfs.subvol_list"] = process.NewInternalProcessFactory(m.SubvolList)
	pm.CmdMap["btrfs.subvol_snapshot"] = process.NewInternalProcessFactory(m.SubvolSnapshot)
}

type btrfsFS struct {
	Label        string        `json:"label"`
	UUID         string        `json:"uuid"`
	TotalDevices int           `json:"total_devices"`
	Used         int64         `json:"used"`
	Devices      []btrfsDevice `json:"devices"`
}

type btrfsDataInfo struct {
	Profile string `json:"profile"`
	Total   int64  `json:"total"`
	Used    int64  `json:"used"`
}

type btrfsFSInfo struct {
	btrfsFS
	Data          btrfsDataInfo `json:"data"`
	System        btrfsDataInfo `json:"system"`
	MetaData      btrfsDataInfo `json:"metadata"`
	GlobalReserve btrfsDataInfo `json:"globalreserve"`
}

type btrfsDevice struct {
	Missing bool   `json:"missing,omitempty"`
	DevID   int    `json:"dev_id"`
	Size    int64  `json:"size"`
	Used    int64  `json:"used"`
	Path    string `json:"path"`
}

type btrfsSubvol struct {
	ID       int
	Gen      int
	TopLevel int
	Path     string
}

var (
	// valid btrfs data & metadata profiles
	Profiles = map[string]struct{}{
		"raid0":  struct{}{},
		"raid1":  struct{}{},
		"raid5":  struct{}{},
		"raid6":  struct{}{},
		"raid10": struct{}{},
		"dup":    struct{}{},
		"single": struct{}{},
		"":       struct{}{},
	}
)

type CreateArgument struct {
	Label     string   `json:"label"`
	Metadata  string   `json:"metadata"`
	Data      string   `json:"data"`
	Devices   []string `json:"devices"`
	Overwrite bool     `json:"overwrite"`
}

type InfoArgument struct {
	Mountpoint string `json:"mountpoint"`
}

type DeviceAddArgument struct {
	InfoArgument
	Devices []string `json:"devices"`
}

type SnapshotArgument struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Readonly    bool   `json:"read_only"`
}

func (arg CreateArgument) Validate() error {
	if len(arg.Devices) == 0 {
		return fmt.Errorf("need to specify devices to create btrfs")
	}
	if v, ok := Profiles[arg.Metadata]; !ok {
		return fmt.Errorf("invalid metadata profile:%v", v)
	}
	if v, ok := Profiles[arg.Data]; !ok {
		return fmt.Errorf("invalid data profile:%v", v)
	}
	return nil
}

func (m *btrfsManager) Create(cmd *core.Command) (interface{}, error) {
	var args CreateArgument
	var opts []string

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	if err := args.Validate(); err != nil {
		return nil, err
	}

	if args.Overwrite {
		opts = append(opts, "-f")
	}
	if args.Label != "" {
		opts = append(opts, "-L", args.Label)
	}
	if args.Metadata != "" {
		opts = append(opts, "-m", args.Metadata)
	}
	if args.Data != "" {
		opts = append(opts, "-d", args.Data)
	}
	opts = append(opts, args.Devices...)

	_, err := pm.GetManager().System("mkfs.btrfs", opts...)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (m *btrfsManager) DeviceAdd(cmd *core.Command) (interface{}, error) {
	var args DeviceAddArgument
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	cmdArgs := []string{"device", "add", "-K", "-f"}
	cmdArgs = append(cmdArgs, args.Devices...)
	cmdArgs = append(cmdArgs, args.Mountpoint)
	_, err := m.btrfs(cmdArgs...)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (m *btrfsManager) DeviceRemove(cmd *core.Command) (interface{}, error) {
	var args DeviceAddArgument
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	cmdArgs := []string{"device", "remove"}
	cmdArgs = append(cmdArgs, args.Devices...)
	cmdArgs = append(cmdArgs, args.Mountpoint)
	_, err := m.btrfs(cmdArgs...)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (m *btrfsManager) list(cmd *core.Command, args []string) ([]btrfsFS, error) {
	defaultargs := []string{"filesystem", "show", "--raw"}
	defaultargs = append(defaultargs, args...)
	result, err := m.btrfs(defaultargs...)
	if err != nil {
		return nil, err
	}

	fss, err := m.parseList(result.Streams.Stdout())
	if err != nil {
		return nil, err
	}

	if fss == nil {
		fss = make([]btrfsFS, 0)
	}

	return fss, err
}

// list btrfs FSs
func (m *btrfsManager) List(cmd *core.Command) (interface{}, error) {
	return m.list(cmd, []string{})
}

// get btrfs info
func (m *btrfsManager) Info(cmd *core.Command) (interface{}, error) {
	var args InfoArgument
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	fss, err := m.list(cmd, []string{args.Mountpoint})
	if err != nil {
		return nil, err
	}

	result, err := m.btrfs("filesystem", "df", "--raw", args.Mountpoint)
	if err != nil {
		return nil, err
	}
	fsinfo := btrfsFSInfo{
		btrfsFS: fss[0],
	}
	err = m.parseFilesystemDF(result.Streams.Stdout(), &fsinfo)
	return fsinfo, err

}

type SubvolArgument struct {
	Path string `json:"path"`
}

type SubvolQuotaArgument struct {
	SubvolArgument
	Limit string `json:"limit"`
}

// create subvolume under a mount point
func (m *btrfsManager) SubvolCreate(cmd *core.Command) (interface{}, error) {
	var args SubvolArgument

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	if args.Path == "" || !strings.HasPrefix(args.Path, "/") {
		return nil, fmt.Errorf("invalid path=%v", args.Path)
	}

	_, err := m.btrfs("subvolume", "create", args.Path)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// delete subvolume under a mount point
func (m *btrfsManager) SubvolDelete(cmd *core.Command) (interface{}, error) {
	var args SubvolArgument

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	if args.Path == "" || !strings.HasPrefix(args.Path, "/") {
		return nil, fmt.Errorf("invalid path=%v", args.Path)
	}

	_, err := m.btrfs("subvolume", "delete", args.Path)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// create quota for a subvolume
func (m *btrfsManager) SubvolQuota(cmd *core.Command) (interface{}, error) {
	var args SubvolQuotaArgument

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	if args.Path == "" || !strings.HasPrefix(args.Path, "/") {
		return nil, fmt.Errorf("invalid path=%v", args.Path)
	}

	_, err := m.btrfs("quota", "enable", args.Path)
	if err != nil {
		return nil, err
	}

	_, err = m.btrfs("qgroup", "limit", args.Limit, args.Path)

	if err != nil {
		return nil, err
	}

	return nil, nil
}

// make a subvol snapshot
func (m *btrfsManager) SubvolSnapshot(cmd *core.Command) (interface{}, error) {
	var args SnapshotArgument
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	in := []string{"subvolume", "snapshot"}
	if args.Readonly {
		in = append(in, "-r")
	}
	in = append(in, args.Source, args.Destination)

	_, err := m.btrfs(in...)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// list subvolume under a mount point
func (m *btrfsManager) SubvolList(cmd *core.Command) (interface{}, error) {
	var args SubvolArgument

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	if args.Path == "" || !strings.HasPrefix(args.Path, "/") {
		return nil, fmt.Errorf("invalid path=%v", args.Path)
	}

	result, err := m.btrfs("subvolume", "list", args.Path)
	if err != nil {
		return nil, err
	}

	return m.parseSubvolList(result.Streams.Stdout())
}

func (m *btrfsManager) btrfs(args ...string) (*core.JobResult, error) {
	return pm.GetManager().System("btrfs", args...)
}

func (m *btrfsManager) parseSubvolList(out string) ([]btrfsSubvol, error) {
	var svs []btrfsSubvol

	for _, line := range strings.Split(out, "\n") {
		var sv btrfsSubvol
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, err := fmt.Sscanf(line, "ID %d gen %d top level %d path %s", &sv.ID, &sv.Gen, &sv.TopLevel, &sv.Path); err != nil {
			return svs, err
		}
		svs = append(svs, sv)
	}
	return svs, nil
}

func (m *btrfsManager) parseFilesystemDF(output string, fsinfo *btrfsFSInfo) error {
	var err error
	lines := reBtrfsFilesystemDf.FindAllStringSubmatch(output, -1)
	for _, line := range lines {
		name := line[1]
		var datainfo *btrfsDataInfo
		switch name {
		case "Data":
			datainfo = &fsinfo.Data
		case "System":
			datainfo = &fsinfo.System
		case "Metadata":
			datainfo = &fsinfo.MetaData
		case "GlobalReserve":
			datainfo = &fsinfo.GlobalReserve
		default:
			continue
		}
		datainfo.Profile = line[2]
		datainfo.Total, err = strconv.ParseInt(line[3], 10, 64)
		if err != nil {
			return err
		}
		datainfo.Used, err = strconv.ParseInt(line[4], 10, 64)
		if err != nil {
			return err
		}

	}

	return err
}

// parse `btrfs filesystem show` output
func (m *btrfsManager) parseList(output string) ([]btrfsFS, error) {
	var fss []btrfsFS

	all := strings.Split(output, "\n")
	if len(all) < 3 {
		return fss, nil
	}

	var fsLines []string
	for i, line := range all {
		line = strings.TrimSpace(line)

		// there are 3 markers of a filesystem
		// - empty line (original btrfs command)
		// - line started with `Label` and not first line (PM wrapped command)
		// - last line (original btrfs command & PM wrapped command)
		if (strings.HasPrefix(line, "Label") && i != 0) || line == "" || i == len(all)-1 {
			if !strings.HasPrefix(line, "Label") {
				fsLines = append(fsLines, line)
			}
			if len(fsLines) < 3 {
				continue
			}
			fs, err := m.parseFS(fsLines)
			if err != nil {
				return fss, err
			}
			fss = append(fss, fs)

			fsLines = []string{}
			if strings.HasPrefix(line, "Label") {
				fsLines = append(fsLines, line)
			}
		} else {
			fsLines = append(fsLines, line)
		}
	}
	return fss, nil
}

func (m *btrfsManager) parseFS(lines []string) (btrfsFS, error) {
	// first line should be label && uuid
	var label, uuid string
	_, err := fmt.Sscanf(lines[0], `Label: %s uuid: %s`, &label, &uuid)
	if err != nil {
		return btrfsFS{}, err
	}
	if label != "none" {
		label = label[1 : len(label)-1]
	}

	// total device & byte used
	var totDevice int
	var used int64
	if _, err := fmt.Sscanf(lines[1], "Total devices %d FS bytes used %d", &totDevice, &used); err != nil {
		return btrfsFS{}, err
	}

	devs, err := m.parseDevices(lines[2:])
	if err != nil {
		return btrfsFS{}, err
	}
	return btrfsFS{
		Label:        label,
		UUID:         uuid,
		TotalDevices: totDevice,
		Used:         used,
		Devices:      devs,
	}, nil
}

func (m *btrfsManager) parseDevices(lines []string) ([]btrfsDevice, error) {
	var devs []btrfsDevice
	for _, line := range lines {
		if line == "" {
			continue
		}
		var dev btrfsDevice
		if _, err := fmt.Sscanf(line, "devid    %d size %d used %d path %s", &dev.DevID, &dev.Size, &dev.Used, &dev.Path); err == nil {
			devs = append(devs, dev)
		}
	}
	return devs, nil
}
