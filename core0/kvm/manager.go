package kvm

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/url"
	//"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/pborman/uuid"
	"github.com/vishvananda/netlink"
)

type kvmManager struct{}

var (
	pattern = regexp.MustCompile(`^\s*(\d+)(.+)\s(\w+)$`)
)

const (
	kvmCreateCommand  = "kvm.create"
	kvmDestroyCommand = "kvm.destroy"
	kvmListCommand    = "kvm.list"
)

func KVMSubsystem() error {
	mgr := &kvmManager{}

	if err := mgr.init(); err != nil {
		return err
	}

	pm.CmdMap[kvmCreateCommand] = process.NewInternalProcessFactory(mgr.create)
	pm.CmdMap[kvmDestroyCommand] = process.NewInternalProcessFactory(mgr.destroy)
	pm.CmdMap[kvmListCommand] = process.NewInternalProcessFactory(mgr.list)

	return nil
}

type CreateParams struct {
	Name   string   `json:"name"`
	CPU    int      `json:"cpu"`
	Memory int      `json:"memory"`
	Images []string `json:"images"`
	Bridge []string `json:"bridge"`
}

func (m *kvmManager) init() error {
	//create default bridge here.
	return nil
}

func (m *kvmManager) mkNBDDisk(u *url.URL, target string) DiskDevice {
	name := strings.Trim(u.Path, "/")

	switch u.Scheme {
	case "nbd":
		fallthrough
	case "nbd+tcp":
		port := u.Port()
		if port == "" {
			port = "10809"
		}
		return DiskDevice{
			Type:   DiskTypeNetwork,
			Device: DiskDeviceTypeDisk,
			Target: DiskTarget{
				Dev: target,
				Bus: "virtio",
			},
			Source: DiskSourceNetwork{
				Protocol: "nbd",
				Name:     name,
				Host: DiskSourceNetworkHost{
					Transport: "tcp",
					Port:      port,
					Name:      u.Hostname(),
				},
			},
		}
	case "nbd+unix":
		return DiskDevice{
			Type:   DiskTypeNetwork,
			Device: DiskDeviceTypeDisk,
			Target: DiskTarget{
				Dev: target,
				Bus: "virtio",
			},
			Source: DiskSourceNetwork{
				Protocol: "nbd",
				Name:     name,
				Host: DiskSourceNetworkHost{
					Transport: "unix",
					Socket:    u.Query().Get("socket"),
				},
			},
		}
	default:
		panic(fmt.Errorf("invalid nbd url: %s", u))
	}
}

func (m *kvmManager) mkDisk(img string, target string) DiskDevice {
	u, err := url.Parse(img)

	if err == nil && strings.Index(u.Scheme, "nbd") == 0 {
		return m.mkNBDDisk(u, target)
	}

	//default fall back to image disk
	return DiskDevice{
		Type:   DiskTypeFile,
		Device: DiskDeviceTypeDisk,
		Target: DiskTarget{
			Dev: target,
			Bus: "ide",
		},
		Source: DiskSourceFile{
			File: img,
		},
	}
}

func (m *kvmManager) create(cmd *core.Command) (interface{}, error) {
	var params CreateParams
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}

	domain := Domain{
		Type: DomainTypeKVM,
		Name: params.Name,
		UUID: uuid.New(),
		Memory: Memory{
			Capacity: params.Memory,
			Unit:     "MB",
		},
		VCPU: params.CPU,
		OS: OS{
			Type: OSType{
				Type: OSTypeTypeHVM,
				Arch: ArchX86_64,
			},
		},
		Devices: Devices{
			Emulator: "/usr/bin/qemu-system-x86_64",
			Devices: []Device{
				SerialDevice{
					Type: SerialDeviceTypePTY,
					Source: SerialSource{
						Path: "/dev/pts/1",
					},
					Target: SerialTarget{
						Port: 0,
					},
					Alias: SerialAlias{
						Name: "serial0",
					},
				},
				ConsoleDevice{
					Type: SerialDeviceTypePTY,
					TTY:  "/dev/pts/1",
					Source: SerialSource{
						Path: "/dev/pts/1",
					},
					Target: ConsoleTarget{
						Port: 0,
						Type: "serial",
					},
					Alias: SerialAlias{
						Name: "serial0",
					},
				},
				GraphicsDevice{
					Type:   GraphicsDeviceTypeVNC,
					Port:   -1,
					KeyMap: "en-us",
					Listen: Listen{
						Type:    "address",
						Address: "0.0.0.0",
					},
				},
			},
		},
	}

	for _, bridge := range params.Bridge {
		_, err := netlink.LinkByName(bridge)
		if err != nil {
			return nil, fmt.Errorf("bridge '%s' not found", bridge)
		}

		domain.Devices.Devices = append(domain.Devices.Devices, InterfaceDevice{
			Type: InterfaceDeviceTypeBridge,
			Source: InterfaceDeviceSourceBridge{
				Bridge: bridge,
			},
			Model: InterfaceDeviceModel{
				Type: "virtio",
			},
		})
	}

	for idx, image := range params.Images {
		target := "vd" + string(97+idx)
		domain.Devices.Devices = append(domain.Devices.Devices, m.mkDisk(image, target))
	}

	data, err := xml.MarshalIndent(domain, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to generate domain xml: %s", err)
	}

	tmp, err := ioutil.TempFile("/tmp", "kvm-domain")
	if err != nil {
		return nil, err
	}
	//defer os.Remove(tmp.Name())
	defer tmp.Close()

	if _, err := tmp.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write domain xml: %s", err)
	}

	tmp.Close()

	//create domain
	virsh := &core.Command{
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: "virsh",
				Args: []string{
					"create", tmp.Name(),
				},
			},
		),
	}
	runner, err := pm.GetManager().RunCmd(virsh)
	if err != nil {
		return nil, fmt.Errorf("failed to start virsh: %s", err)
	}
	result := runner.Wait()
	if result.State != core.StateSuccess {
		return nil, fmt.Errorf(result.Streams[1])
	}

	return nil, nil
}

type DestroyParams struct {
	Name string `json:"name"`
}

func (m *kvmManager) destroy(cmd *core.Command) (interface{}, error) {
	var params DestroyParams
	if err := json.Unmarshal(*cmd.Arguments, &params); err != nil {
		return nil, err
	}
	virsh := &core.Command{
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: "virsh",
				Args: []string{
					"destroy", params.Name,
				},
			},
		),
	}
	runner, err := pm.GetManager().RunCmd(virsh)
	if err != nil {
		return nil, fmt.Errorf("failed to destroy machine: %s", err)
	}
	result := runner.Wait()
	if result.State != core.StateSuccess {
		return nil, fmt.Errorf(result.Streams[1])
	}
	return nil, nil
}

type Machine struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

func (m *kvmManager) list(cmd *core.Command) (interface{}, error) {
	virsh := &core.Command{
		ID:      uuid.New(),
		Command: process.CommandSystem,
		Arguments: core.MustArguments(
			process.SystemCommandArguments{
				Name: "virsh",
				Args: []string{
					"list", "--all",
				},
			},
		),
	}
	runner, err := pm.GetManager().RunCmd(virsh)
	if err != nil {
		return nil, fmt.Errorf("failed to destroy machine: %s", err)
	}
	result := runner.Wait()
	if result.State != core.StateSuccess {
		return nil, fmt.Errorf(result.Streams[1])
	}

	out := result.Streams[0]

	found := make([]Machine, 0)
	lines := strings.Split(out, "\n")
	if len(lines) <= 3 {
		return found, nil
	}

	lines = lines[2:]

	for _, line := range lines {
		match := pattern.FindStringSubmatch(line)
		if len(match) != 4 {
			continue
		}
		id, _ := strconv.ParseInt(match[1], 10, 32)
		found = append(found, Machine{
			ID:    int(id),
			Name:  strings.TrimSpace(match[2]),
			State: strings.TrimSpace(match[3]),
		})
	}

	return found, nil
}
