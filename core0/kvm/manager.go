package kvm

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/g8os/core0/base/pm/process"
	"github.com/pborman/uuid"
	"github.com/vishvananda/netlink"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
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

func init() {
	mgr := &kvmManager{}

	pm.CmdMap[kvmCreateCommand] = process.NewInternalProcessFactory(mgr.create)
	pm.CmdMap[kvmDestroyCommand] = process.NewInternalProcessFactory(mgr.destroy)
	pm.CmdMap[kvmListCommand] = process.NewInternalProcessFactory(mgr.list)
}

type CreateParams struct {
	Name   string `json:"name"`
	CPU    int    `json:"cpu"`
	Memory int    `json:"memory"`
	Image  string `json:"image"`
	Bridge string `json:"bridge"`
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
				DiskDevice{
					Type:   DiskTypeFile,
					Device: DiskDeviceTypeDisk,
					Target: DiskTarget{
						Dev: "hda",
						Bus: "ide",
					},
					Source: DiskSourceFile{
						File: params.Image,
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

	if params.Bridge != "" {
		_, err := netlink.LinkByName(params.Bridge)
		if err != nil {
			return nil, fmt.Errorf("bridge '%s' not found", params.Bridge)
		}

		domain.Devices.Devices = append(domain.Devices.Devices, InterfaceDevice{
			Type: InterfaceDeviceTypeBridge,
			Source: InterfaceDeviceSourceBridge{
				Bridge: params.Bridge,
			},
		})
	}

	data, err := xml.MarshalIndent(domain, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to generate domain xml: %s", err)
	}

	tmp, err := ioutil.TempFile("/tmp", "kvm-domain")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
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
