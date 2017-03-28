package builtin

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pborman/uuid"

	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
)

var (
	errBtrfsNoFS        = errors.New("No btrfs FS found")
	reBtrfsFilesystemDf = regexp.MustCompile(`(?m:(\w+),\s(\w+):\s+total=(\d+),\s+used=(\d+))`)
)

func init() {
	pm.CmdMap["btrfs.list"] = process.NewInternalProcessFactory(btrfsList)
	pm.CmdMap["btrfs.info"] = process.NewInternalProcessFactory(btrfsInfo)
	pm.CmdMap["btrfs.create"] = process.NewInternalProcessFactory(btrfsCreate)
	pm.CmdMap["btrfs.add_device"] = process.NewInternalProcessFactory(btrfsAddDevice)
	pm.CmdMap["btrfs.remove_device"] = process.NewInternalProcessFactory(btrfsRemoveDevice)
	pm.CmdMap["btrfs.subvol_create"] = process.NewInternalProcessFactory(btrfsSubvolCreate)
	pm.CmdMap["btrfs.subvol_delete"] = process.NewInternalProcessFactory(btrfsSubvolDelete)
	pm.CmdMap["btrfs.subvol_list"] = process.NewInternalProcessFactory(btrfsSubvolList)
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

var (
	// valid btrfs data & metadata profiles
	btrfsProfiles = map[string]struct{}{
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

type btrfsCreateArgument struct {
	Label    string   `json:"label"`
	Metadata string   `json:"metadata"`
	Data     string   `json:"data"`
	Devices  []string `json:"devices"`
}

type btrfsInfoArgument struct {
	Mountpoint string `json:"mountpoint"`
}

type btrfsAddDevicesArgument struct {
	btrfsInfoArgument
	Devices []string `json:"devices"`
}

type btrfsSubvol struct {
	ID       int
	Gen      int
	TopLevel int
	Path     string
}

func (arg btrfsCreateArgument) Validate() error {
	if len(arg.Devices) == 0 {
		return fmt.Errorf("need to specify devices to create btrfs")
	}
	if v, ok := btrfsProfiles[arg.Metadata]; !ok {
		return fmt.Errorf("invalid metadata profile:%v", v)
	}
	if v, ok := btrfsProfiles[arg.Data]; !ok {
		return fmt.Errorf("invalid data profile:%v", v)
	}
	return nil
}

func btrfsCreate(cmd *core.Command) (interface{}, error) {
	var args btrfsCreateArgument
	var opts []string

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	if err := args.Validate(); err != nil {
		return nil, err
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

	result, err := runBtrfsCmd("mkfs.btrfs", opts)
	if err != nil {
		return nil, err
	}

	if result.State != core.StateSuccess {
		return "", fmt.Errorf("error creating btrfs filesystem: %v", result.Streams)
	}

	return nil, nil
}

func btrfsAddDevice(cmd *core.Command) (interface{}, error) {
	var args btrfsAddDevicesArgument
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	cmdArgs := []string{"device", "add", "-K", "-f"}
	cmdArgs = append(cmdArgs, args.Devices...)
	cmdArgs = append(cmdArgs, args.Mountpoint)
	result, err := runBtrfsCmd("btrfs", cmdArgs)
	if err != nil {
		return nil, err
	}

	if result.State != core.StateSuccess {
		return nil, fmt.Errorf("%v", result.Streams)
	}

	return nil, nil
}

func btrfsRemoveDevice(cmd *core.Command) (interface{}, error) {
	var args btrfsAddDevicesArgument
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	cmdArgs := []string{"device", "remove"}
	cmdArgs = append(cmdArgs, args.Devices...)
	cmdArgs = append(cmdArgs, args.Mountpoint)
	result, err := runBtrfsCmd("btrfs", cmdArgs)
	if err != nil {
		return nil, err
	}

	if result.State != core.StateSuccess {
		return nil, fmt.Errorf("%v", result.Streams)
	}

	return nil, nil
}

func btrfsListCmd(cmd *core.Command, args []string) ([]btrfsFS, error) {
	defaultargs := []string{"filesystem", "show", "--raw"}
	result, err := runBtrfsCmd("btrfs", append(defaultargs, args...))
	if err != nil {
		return nil, err
	}

	if result.State != core.StateSuccess || len(result.Streams) == 0 {
		return nil, fmt.Errorf("error listing btrfs filesystem: %v", result.Streams)
	}
	fss, err := btrfsParseList(result.Streams[0])
	if err != nil {
		return nil, err
	}

	if fss == nil {
		fss = make([]btrfsFS, 0)
	}

	return fss, err
}

// list btrfs FSs
func btrfsList(cmd *core.Command) (interface{}, error) {
	return btrfsListCmd(cmd, []string{})
}

// get btrfs info
func btrfsInfo(cmd *core.Command) (interface{}, error) {
	var args btrfsInfoArgument
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	fss, err := btrfsListCmd(cmd, []string{args.Mountpoint})
	if err != nil {
		return nil, err
	}

	result, err := runBtrfsCmd("btrfs", []string{"filesystem", "df", "--raw", args.Mountpoint})
	if err != nil {
		return nil, err
	}
	fsinfo := btrfsFSInfo{
		btrfsFS: fss[0],
	}
	err = btrfsParserFilesystemDF(result.Streams[0], &fsinfo)
	return fsinfo, err

}

type btrfsSubvolArgument struct {
	Path string `json:"path"`
}

// create subvolume under a mount point
func btrfsSubvolCreate(cmd *core.Command) (interface{}, error) {
	var args btrfsSubvolArgument

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	if args.Path == "" || !strings.HasPrefix(args.Path, "/") {
		return nil, fmt.Errorf("invalid path=%v", args.Path)
	}

	result, err := runBtrfsCmd("btrfs", []string{"subvolume", "create", args.Path})
	if err != nil {
		return nil, err
	}

	if result.State != core.StateSuccess {
		return nil, fmt.Errorf("error creating btrfs subvolume: %v:%v", result.Streams, result.Data)
	}
	return nil, nil
}

// delete subvolume under a mount point
func btrfsSubvolDelete(cmd *core.Command) (interface{}, error) {
	var args btrfsSubvolArgument

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	if args.Path == "" || !strings.HasPrefix(args.Path, "/") {
		return nil, fmt.Errorf("invalid path=%v", args.Path)
	}

	result, err := runBtrfsCmd("btrfs", []string{"subvolume", "delete", args.Path})
	if err != nil {
		return nil, err
	}

	if result.State != core.StateSuccess {
		return nil, fmt.Errorf("error deleting btrfs subvolume: %v:%v", result.Streams, result.Data)
	}

	return nil, nil
}

// list subvolume under a mount point
func btrfsSubvolList(cmd *core.Command) (interface{}, error) {
	var args btrfsSubvolArgument

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	if args.Path == "" || !strings.HasPrefix(args.Path, "/") {
		return nil, fmt.Errorf("invalid path=%v", args.Path)
	}

	result, err := runBtrfsCmd("btrfs", []string{"subvolume", "list", args.Path})
	if err != nil {
		return nil, err
	}

	if result.State != core.StateSuccess || len(result.Streams) != 2 {
		return nil, fmt.Errorf("error list btrfs subvolume: %v:%v", result.Streams, result.Data)
	}

	return btrfsParseSubvolList(result.Streams[0])
}

func runBtrfsCmd(cmd string, args []string) (*core.JobResult, error) {
	shellCmd := &core.Command{
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: cmd,
				Args: args,
			},
		),
	}

	runner, err := pm.GetManager().RunCmd(shellCmd)
	if err != nil {
		return nil, err
	}

	return runner.Wait(), nil
}
func btrfsParseSubvolList(out string) ([]btrfsSubvol, error) {
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

func btrfsParserFilesystemDF(output string, fsinfo *btrfsFSInfo) error {
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
func btrfsParseList(output string) ([]btrfsFS, error) {
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
			fs, err := btrfsParseFS(fsLines)
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

func btrfsParseFS(lines []string) (btrfsFS, error) {
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

	devs, err := btrfsParseDevices(lines[2:])
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

func btrfsParseDevices(lines []string) ([]btrfsDevice, error) {
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
