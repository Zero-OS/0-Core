package builtin

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"

	"strings"

	"github.com/zero-os/0-core/base/pm"
)

/*
Implementation for disk.getinfo
Note that, the rest of the disk extension implementation is done in conf/disk.toml file
*/

var (
	freeSpaceRegex = regexp.MustCompile(`(?m:^\s*(\d+)B\s+(\d+)B\s+(\d+)B\s+Free Space$)`)
	partTableRegex = regexp.MustCompile(`Partition Table: (\w+)`)
)

const mountsFile = "/proc/mounts"

type diskMgr struct{}

func init() {
	d := (*diskMgr)(nil)
	pm.RegisterBuiltIn("disk.getinfo", d.info)
	pm.RegisterBuiltIn("disk.list", d.list)
	pm.RegisterBuiltIn("disk.protect", d.protect)
	pm.RegisterBuiltIn("disk.mounts", d.mounts)
	pm.RegisterBuiltIn("disk.smartctl-info", d.smartctlInfo)
	pm.RegisterBuiltIn("disk.smartctl-health", d.smartctlHealth)
}

type diskInfo struct {
	Disk string `json:"disk"`
	Part string `json:"part"`
}

type DiskFreeBlock struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
	Size  uint64 `json:"size"`
}

type DiskInfoResult struct {
	lsblkResult
	Start     uint64          `json:"start"`
	End       uint64          `json:"end"`
	Size      uint64          `json:"size"`
	BlockSize uint64          `json:"blocksize"`
	Table     string          `json:"table"`
	Free      []DiskFreeBlock `json:"free"`
}

type lsblkListResult struct {
	BlockDevices []lsblkResult `json:"blockdevices"`
}

type lsblkResult struct {
	Name       string        `json:"name"`
	Kname      string        `json:"kname"`
	MajMin     string        `json:"maj:min"`
	Fstype     interface{}   `json:"fstype"`
	Mountpoint interface{}   `json:"mountpoint"`
	Label      interface{}   `json:"label"`
	UUID       interface{}   `json:"uuid"`
	Parttype   interface{}   `json:"parttype"`
	Partlabel  interface{}   `json:"partlabel"`
	Partuuid   interface{}   `json:"partuuid"`
	Partflags  interface{}   `json:"partflags"`
	Ra         string        `json:"ra"`
	Ro         string        `json:"ro"`
	Rm         string        `json:"rm"`
	Hotplug    string        `json:"hotplug"`
	Model      string        `json:"model"`
	Serial     string        `json:"serial"`
	Size       string        `json:"size"`
	State      interface{}   `json:"state"`
	Owner      string        `json:"owner"`
	Group      string        `json:"group"`
	Mode       string        `json:"mode"`
	Alignment  string        `json:"alignment"`
	MinIo      string        `json:"min-io"`
	OptIo      string        `json:"opt-io"`
	PhySec     string        `json:"phy-sec"`
	LogSec     string        `json:"log-sec"`
	Rota       string        `json:"rota"`
	Sched      interface{}   `json:"sched"`
	RqSize     string        `json:"rq-size"`
	Type       string        `json:"type"`
	DiscAln    string        `json:"disc-aln"`
	DiscGran   string        `json:"disc-gran"`
	DiscMax    string        `json:"disc-max"`
	DiscZero   string        `json:"disc-zero"`
	Wsame      string        `json:"wsame"`
	Wwn        interface{}   `json:"wwn"`
	Rand       string        `json:"rand"`
	Pkname     interface{}   `json:"pkname"`
	Hctl       interface{}   `json:"hctl"`
	Tran       string        `json:"tran"`
	Subsystems string        `json:"subsystems"`
	Rev        interface{}   `json:"rev"`
	Vendor     interface{}   `json:"vendor"`
	Children   []lsblkResult `json:"children,omitempty"`
}

type diskMount struct {
	Mountpoint string            `json:"mountpoint"`
	Filesystem string            `json:"filesystem"`
	Options    map[string]string `json:"options"`
}

func (d *diskMgr) readUInt64(p string) (uint64, error) {
	bytes, err := ioutil.ReadFile(p)
	if err != nil {
		return 0, err
	}
	var r uint64
	if _, err := fmt.Sscanf(string(bytes), "%d", &r); err != nil {
		return 0, err
	}

	return r, nil
}

func (d *diskMgr) lsblk(dev string) (*lsblkResult, error) {
	result, err := pm.System("lsblk", "-O", "-J", fmt.Sprintf("/dev/%s", dev))

	if err != nil {
		return nil, err
	}

	cmdOutput := struct {
		BlockDevices []lsblkResult `json:"blockdevices"`
	}{}
	if err := json.Unmarshal([]byte(result.Streams.Stdout()), &cmdOutput); err != nil {
		return nil, err
	}

	if len(cmdOutput.BlockDevices) >= 1 {
		return &cmdOutput.BlockDevices[0], nil

	}

	return nil, fmt.Errorf("not device with the name /dev/%s", dev)

}
func (d *diskMgr) blockSize(dev string) (uint64, error) {
	return d.readUInt64(fmt.Sprintf("/sys/block/%s/queue/logical_block_size", dev))
}

func (d *diskMgr) getTableInfo(disk string) (string, []DiskFreeBlock, error) {
	blocks := make([]DiskFreeBlock, 0)
	result, err := pm.System("parted", fmt.Sprintf("/dev/%s", disk), "unit", "B", "print", "free")

	if err != nil {
		return "", blocks, err
	}

	table := ""
	tableMatch := partTableRegex.FindStringSubmatch(result.Streams.Stdout())
	if len(tableMatch) == 2 {
		table = tableMatch[1]
	}

	matches := freeSpaceRegex.FindAllStringSubmatch(result.Streams.Stdout(), -1)
	for _, match := range matches {
		bstart, _ := strconv.ParseUint(match[1], 10, 64)
		bend, _ := strconv.ParseUint(match[2], 10, 64)
		bsize, _ := strconv.ParseUint(match[3], 10, 64)

		blocks = append(blocks, DiskFreeBlock{
			Start: bstart,
			End:   bend,
			Size:  bsize,
		})
	}

	return table, blocks, nil
}

func (d *diskMgr) diskInfo(disk string) (*DiskInfoResult, error) {

	var info DiskInfoResult

	lsblk, err := d.lsblk(disk)
	if err != nil {
		return nil, err
	}
	info.lsblkResult = *lsblk

	bs, err := d.blockSize(disk)
	if err != nil {
		return nil, err
	}

	size, err := d.readUInt64(fmt.Sprintf("/sys/block/%s/size", disk))
	if err != nil {
		return nil, err
	}
	info.Size = size * bs
	info.End = (size * bs) - 1

	info.BlockSize = bs
	//get free blocks.
	table, blocks, err := d.getTableInfo(disk)
	if err != nil {
		return nil, err
	}
	info.Table = table
	info.Free = blocks

	return &info, nil
}

func (d *diskMgr) partInfo(disk, part string) (*DiskInfoResult, error) {
	var info DiskInfoResult

	lsblk, err := d.lsblk(part)
	if err != nil {
		return nil, err
	}
	info.lsblkResult = *lsblk

	bs, err := d.blockSize(disk)
	if err != nil {
		return nil, err
	}

	start, err := d.readUInt64(fmt.Sprintf("/sys/block/%s/%s/start", disk, part))
	if err != nil {
		return nil, err
	}

	size, err := d.readUInt64(fmt.Sprintf("/sys/block/%s/%s/size", disk, part))
	if err != nil {
		return nil, err
	}

	info.Start = start * bs
	info.Size = size * bs
	info.End = info.Start + info.Size - 1

	info.BlockSize = bs
	info.Free = make([]DiskFreeBlock, 0) //this is just to make the return consistent
	return &info, nil
}

func (d *diskMgr) info(cmd *pm.Command) (interface{}, error) {
	var args diskInfo

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	if args.Part == "" {
		return d.diskInfo(args.Disk)
	}

	return d.partInfo(args.Disk, args.Part)
}

func (d *diskMgr) smartctlInfo(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Device string `json:"device"`
	}

	var info struct {
		Model                 string `json:"model"`
		SerialNumber          string `json:"serial_number"`
		DeviceID              string `json:"device_id"`
		FirmwareVersion       string `json:"firmware_version"`
		UserCapacity          int    `json:"user_capacity"`
		SectorSize            int    `json:"sector_size"`
		RotationRate          string `json:"rotation_rate"`
		Device                string `json:"device"`
		ATAVersion            string `json:"ata_version"`
		SATAVersion           string `json:"sata_version"`
		SmartSupportAvailable bool   `json:"smart_support_available"`
		SmartSupportEnabled   bool   `json:"smart_support_enabled"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	result, err := pm.System("smartctl", "-i", args.Device)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(result.Streams.Stdout(), "\n")

	for _, line := range lines {
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.Split(line, ":")
		value := strings.TrimSpace(parts[1])
		switch parts[0] {
		case "Device Model":
			info.Model = value
		case "Serial Number":
			info.SerialNumber = value
		case "LU WWN Device Id":
			info.DeviceID = value
		case "Firmware Version":
			info.FirmwareVersion = value
		case "User Capacity":
			sizeBytes := strings.Split(value, "bytes")[0]
			info.UserCapacity, err = strconv.Atoi(strings.Replace(strings.TrimSpace(sizeBytes), ",", "", -1))
		case "Sector Size":
			sizeBytes := strings.Split(value, "bytes")[0]
			info.SectorSize, err = strconv.Atoi(strings.Replace(strings.TrimSpace(sizeBytes), ",", "", -1))
		case "Rotation Rate":
			info.RotationRate = value
		case "Device is":
			info.Device = value
		case "ATA Version is":
			info.ATAVersion = value
		case "SATA Version is":
			info.SATAVersion = value
		case "SMART support is":
			if strings.Contains(value, "Available") {
				info.SmartSupportAvailable = true
			}
			if strings.Contains(value, "Enabled") {
				info.SmartSupportEnabled = true
			}
		}
	}
	return info, nil
}

func (d *diskMgr) smartctlHealth(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Device string `json:"device"`
	}

	var health struct {
		Passed bool `json:"passed"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	result, err := pm.System("smartctl", "-H", args.Device)
	if err != nil {
		return nil, err
	}
	if strings.Contains(result.Streams.Stdout(), "PASSED") {
		health.Passed = true
	}
	return health, nil
}

func (d *diskMgr) list(cmd *pm.Command) (interface{}, error) {
	result, err := pm.System("lsblk", "--json", "--output-all", "--bytes", "--exclude", "1,2")
	if err != nil {
		return nil, err
	}

	var disks lsblkListResult
	if err := json.Unmarshal([]byte(result.Streams.Stdout()), &disks); err != nil {
		return nil, err
	}

	parentDiskName := ""
	ret := []*DiskInfoResult{}
	for _, disk := range disks.BlockDevices {
		diskInfo := DiskInfoResult{
			lsblkResult: disk,
		}

		diskInfo.BlockSize, err = d.blockSize(disk.Name)
		if err != nil {
			return nil, err
		}

		if disk.Type == "disk" {
			parentDiskName = disk.Name

			size, err := d.readUInt64(fmt.Sprintf("/sys/block/%s/size", disk.Name))
			if err != nil {
				return nil, err
			}
			diskInfo.Size = size * diskInfo.BlockSize
			diskInfo.End = (size * diskInfo.BlockSize) - 1

			//get free blocks.
			table, blocks, err := d.getTableInfo(disk.Name)
			if err != nil {
				return nil, err
			}
			diskInfo.Table = table
			diskInfo.Free = blocks

		} else if disk.Type == "part" {

			start, err := d.readUInt64(fmt.Sprintf("/sys/block/%s/%s/start", parentDiskName, disk.Name))
			if err != nil {
				return nil, err
			}

			size, err := d.readUInt64(fmt.Sprintf("/sys/block/%s/%s/size", parentDiskName, disk.Name))
			if err != nil {
				return nil, err
			}

			diskInfo.Start = start * diskInfo.BlockSize
			diskInfo.Size = size * diskInfo.BlockSize
			diskInfo.End = diskInfo.Start + diskInfo.Size - 1

			diskInfo.Free = make([]DiskFreeBlock, 0) //this is just to make the return consistent
		}
		ret = append(ret, &diskInfo)
	}

	return ret, nil
}

func (d *diskMgr) mounts(cmd *pm.Command) (interface{}, error) {
	file, err := ioutil.ReadFile(mountsFile)
	if err != nil {
		return nil, err
	}
	return parseMountCmd(string(file)), nil

}

func parseMountCmd(mount string) map[string][]diskMount {
	mountpoints := make(map[string][]diskMount)

	lines := strings.Split(mount, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, " ")
		device := parts[0]
		mountpoint := parts[1]
		fs := parts[2]

		optionsMap := make(map[string]string)
		optionsList := strings.Split(parts[3], ",")

		for _, option := range optionsList {
			optionList := strings.Split(option, "=")
			if len(optionList) == 1 {
				optionsMap[optionList[0]] = "1"
			} else {
				optionsMap[optionList[0]] = optionList[1]
			}
		}

		mount := diskMount{
			mountpoint,
			fs,
			optionsMap,
		}
		mountpoints[device] = append(mountpoints[device], mount)

	}

	return mountpoints
}

func (d *diskMgr) protect(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Partuuid string `json:"partuuid"`
	}

	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	if args.Partuuid == "" {
		return nil, fmt.Errorf("partuuid is required")
	}

	result, err := d.list(nil)
	if err != nil {
		return nil, err
	}

	list, ok := result.(lsblkListResult)
	if !ok {
		return nil, fmt.Errorf("unexpected return from list!")
	}

	for _, dev := range list.BlockDevices {
		for _, part := range dev.Children {
			if part.Partuuid != args.Partuuid {
				continue
			}
			//protect the partition and protect the device
			if _, err := pm.System("blockdev", "--setro", fmt.Sprintf("/dev/%s", part.Name)); err != nil {
				return nil, err
			}

			if _, err := pm.System("blockdev", "--setro", fmt.Sprintf("/dev/%s", dev.Name)); err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
}
