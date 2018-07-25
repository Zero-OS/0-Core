package btrfs

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	logging "github.com/op/go-logging"
	"github.com/zero-os/0-core/base/pm"
)

var (
	log = logging.MustGetLogger("btrfs")

	reBtrfsFilesystemDf = regexp.MustCompile(`(?m:(\w+),\s(\w+):\s+total=(\d+),\s+used=(\d+))`)
	reBtrfsQgroup       = regexp.MustCompile(`(?m:^(\d+/\d+)\s+(\d+)\s+(\d+)\s+(\d+|none)\s+(\d+|none).*$)`)
)

type btrfsManager struct{}

func init() {
	var m btrfsManager

	pm.RegisterBuiltIn("btrfs.list", m.List)
	pm.RegisterBuiltIn("btrfs.info", m.Info)
	pm.RegisterBuiltIn("btrfs.create", m.Create)
	pm.RegisterBuiltIn("btrfs.device_add", m.DeviceAdd)
	pm.RegisterBuiltIn("btrfs.device_remove", m.DeviceRemove)
	pm.RegisterBuiltIn("btrfs.subvol_create", m.SubvolCreate)
	pm.RegisterBuiltIn("btrfs.subvol_delete", m.SubvolDelete)
	pm.RegisterBuiltIn("btrfs.subvol_quota", m.SubvolQuota)
	pm.RegisterBuiltIn("btrfs.subvol_list", m.SubvolList)
	pm.RegisterBuiltIn("btrfs.subvol_snapshot", m.SubvolSnapshot)
}

type btrfsFS struct {
	Label        string        `json:"label"`
	UUID         string        `json:"uuid"`
	TotalDevices int           `json:"total_devices"`
	Used         int64         `json:"used"`
	Devices      []btrfsDevice `json:"devices"`
	Warnings     string        `json:"warnings"`
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
	Quota    uint64
}

type btrfsQGroup struct {
	ID      string
	Rfer    uint64
	Excl    uint64
	MaxRfer uint64
	MaxExcl uint64
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

func (m *btrfsManager) Create(cmd *pm.Command) (interface{}, error) {
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

	_, err := pm.System("mkfs.btrfs", opts...)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (m *btrfsManager) DeviceAdd(cmd *pm.Command) (interface{}, error) {
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

func (m *btrfsManager) DeviceRemove(cmd *pm.Command) (interface{}, error) {
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

func (m *btrfsManager) list(cmd *pm.Command, args []string) ([]btrfsFS, error) {
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
func (m *btrfsManager) List(cmd *pm.Command) (interface{}, error) {
	return m.list(cmd, []string{})
}

// get btrfs info
func (m *btrfsManager) Info(cmd *pm.Command) (interface{}, error) {
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
func (m *btrfsManager) SubvolCreate(cmd *pm.Command) (interface{}, error) {
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
func (m *btrfsManager) SubvolDelete(cmd *pm.Command) (interface{}, error) {
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
func (m *btrfsManager) SubvolQuota(cmd *pm.Command) (interface{}, error) {
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

func (m *btrfsManager) parseQGroups(s string) map[string]btrfsQGroup {
	qgroups := make(map[string]btrfsQGroup)
	for _, line := range reBtrfsQgroup.FindAllStringSubmatch(s, -1) {
		qgroup := btrfsQGroup{
			ID: line[1],
		}

		qgroup.Rfer, _ = strconv.ParseUint(line[2], 10, 64)
		qgroup.Excl, _ = strconv.ParseUint(line[3], 10, 64)
		if line[4] != "none" {
			qgroup.MaxRfer, _ = strconv.ParseUint(line[4], 10, 64)
		}

		if line[5] != "none" {
			qgroup.MaxExcl, _ = strconv.ParseUint(line[5], 10, 64)
		}

		qgroups[qgroup.ID] = qgroup
	}

	return qgroups
}

func (m *btrfsManager) getQGroups(path string) (map[string]btrfsQGroup, error) {
	job, err := m.btrfs("qgroup", "show", "-re", "--raw", path)
	if job == nil && err != nil {
		return nil, err
	} else if err != nil {
		msg := job.Streams.Stderr()
		if strings.IndexAny(msg, "No such file or directory") != -1 {
			return nil, nil
		}
	}

	return m.parseQGroups(job.Streams.Stdout()), nil
}

// make a subvol snapshot
func (m *btrfsManager) SubvolSnapshot(cmd *pm.Command) (interface{}, error) {
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
func (m *btrfsManager) SubvolList(cmd *pm.Command) (interface{}, error) {
	var args SubvolArgument

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	if args.Path == "" || !strings.HasPrefix(args.Path, "/") {
		return nil, fmt.Errorf("invalid path=%v", args.Path)
	}

	result, err := m.btrfs("subvolume", "list", "-o", args.Path)
	if err != nil {
		return nil, err
	}

	volumes, err := m.parseSubvolList(result.Streams.Stdout())
	if err != nil {
		return nil, err
	}

	qgroups, err := m.getQGroups(args.Path)
	if err != nil {
		return nil, err
	}

	for i := range volumes {
		volume := &volumes[i]
		group, ok := qgroups[fmt.Sprintf("0/%d", volume.ID)]
		if !ok {
			continue
		}

		volume.Quota = group.MaxRfer
	}

	return volumes, nil
}

func (m *btrfsManager) btrfs(args ...string) (*pm.JobResult, error) {
	log.Debugf("btrfs %v", args)
	return pm.System("btrfs", args...)
}

func (m *btrfsManager) parseSubvolList(out string) ([]btrfsSubvol, error) {
	lines := strings.Split(out, "\n")
	svs := make([]btrfsSubvol, 0, len(lines))

	for _, line := range lines {
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

	blocks := strings.Split(output, "\n\n")
	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}
		// Ensure that fsLines starts with Label (and collect all warnings into fs.Warnings)
		labelIdx := strings.Index(block, "Label:")
		if labelIdx != 0 {
			block = block[labelIdx:]
		}
		fsLines := strings.Split(block, "\n")
		if len(fsLines) < 3 {
			continue
		}
		fs, err := m.parseFS(fsLines)

		if err != nil {
			return fss, err
		}
		fss = append(fss, fs)
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
	if _, err := fmt.Sscanf(strings.TrimSpace(lines[1]), "Total devices %d FS bytes used %d", &totDevice, &used); err != nil {
		return btrfsFS{}, err
	}
	var validDevsLines []string
	var fsWarnings string
	for _, line := range lines[2:] {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "**") {
			// a warning
			fsWarnings += trimmedLine
		} else {
			validDevsLines = append(validDevsLines, line)
		}
	}
	devs, err := m.parseDevices(validDevsLines)
	if err != nil {
		return btrfsFS{}, err
	}
	return btrfsFS{
		Label:        label,
		UUID:         uuid,
		TotalDevices: totDevice,
		Used:         used,
		Devices:      devs,
		Warnings:     fsWarnings,
	}, nil
}

func (m *btrfsManager) parseDevices(lines []string) ([]btrfsDevice, error) {
	var devs []btrfsDevice
	for _, line := range lines {
		if line == "" {
			continue
		}
		var dev btrfsDevice
		if _, err := fmt.Sscanf(strings.TrimSpace(line), "devid    %d size %d used %d path %s", &dev.DevID, &dev.Size, &dev.Used, &dev.Path); err == nil {
			devs = append(devs, dev)
		}
	}
	return devs, nil
}
