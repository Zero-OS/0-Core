package kvm

import (
	"encoding/json"
	"fmt"
	"github.com/g8os/core0/base/pm"
	"github.com/g8os/core0/base/pm/core"
	"github.com/pborman/uuid"
	"github.com/vishvananda/netlink"
	"net"
	"strconv"
	"strings"
)

const (
	OVSTag       = "ovs"
	OVSBackPlane = "backplane"
	OVSVXBackend = "vxbackend"
)

func (m *kvmManager) setNetworking(args *CreateParams, seq uint16, domain *Domain) error {
	for _, nic := range args.Nics {
		switch nic.Type {
		case "default":
			if err := m.setDefaultNetwork(args, seq, domain); err != nil {
				return err
			}
		case "vlan":
		case "vxlan":
			if err := m.setVXLanNetwork(domain, &nic); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported network mode: %s", nic.Type)
		}
	}

	return nil
}

func (m *kvmManager) setVLanNetwork(domain *Domain, nic *Nic) error {
	vlanID, err := strconv.ParseInt(nic.ID, 10, 16)
	if err != nil {
		return err
	}
	if vlanID == 0 || vlanID >= 4095 {
		return fmt.Errorf("invalid vlan id (1-4094)")
	}

	inf := InterfaceDevice{
		Type: InterfaceDeviceTypeBridge,
		Model: InterfaceDeviceModel{
			Type: "virtio",
		},
	}

	if nic.HWAddress != "" {
		hw, err := net.ParseMAC(nic.HWAddress)
		if err != nil {
			return err
		}
		inf.Mac = &InterfaceDeviceMac{
			Address: hw.String(),
		}
	}
	//find the container with OVS tag
	ovs := m.conmgr.GetOneWithTags(OVSTag)
	if ovs == nil {
		return fmt.Errorf("ovs is needed for VLAN network type")
	}

	//ensure that a bridge is available with that vlan tag.
	//we dispatch the ovs.vlan-ensure command to container.
	result, err := m.conmgr.Dispatch(ovs.ID(), &core.Command{
		Command: "ovs.vlan-ensure",
		Arguments: core.MustArguments(map[string]interface{}{
			"master": OVSBackPlane,
			"vlan":   vlanID,
		}),
	})

	if err != nil {
		return err
	}

	if result.State != core.StateSuccess {
		return fmt.Errorf("failed to ensure vlan bridge: %v", result.Data)
	}

	var bridge string
	if err := json.Unmarshal([]byte(result.Data), &bridge); err != nil {
		return fmt.Errorf("failed to load vlan-ensure result: %s", err)
	}

	inf.Source = InterfaceDeviceSourceBridge{
		Bridge: bridge,
	}

	domain.Devices.Devices = append(domain.Devices.Devices, inf)
	return nil
}

func (m *kvmManager) setVXLanNetwork(domain *Domain, nic *Nic) error {
	vxlan, err := strconv.ParseInt(nic.ID, 10, 64)
	if err != nil {
		return err
	}
	inf := InterfaceDevice{
		Type: InterfaceDeviceTypeBridge,
		Model: InterfaceDeviceModel{
			Type: "virtio",
		},
	}
	if nic.HWAddress != "" {
		hw, err := net.ParseMAC(nic.HWAddress)
		if err != nil {
			return err
		}
		inf.Mac = &InterfaceDeviceMac{
			Address: hw.String(),
		}
	}
	//find the container with OVS tag
	ovs := m.conmgr.GetOneWithTags(OVSTag)
	if ovs == nil {
		return fmt.Errorf("ovs is needed for VXLAN network type")
	}

	//ensure that a bridge is available with that vlan tag.
	//we dispatch the ovs.vxlan-ensure command to container.
	result, err := m.conmgr.Dispatch(ovs.ID(), &core.Command{
		Command: "ovs.vxlan-ensure",
		Arguments: core.MustArguments(map[string]interface{}{
			"master": OVSVXBackend,
			"vxlan":  vxlan,
		}),
	})

	if err != nil {
		return err
	}

	if result.State != core.StateSuccess {
		return fmt.Errorf("failed to ensure vlan bridge: %v", result.Data)
	}
	//brname:
	var bridge string
	if err := json.Unmarshal([]byte(result.Data), &bridge); err != nil {
		return fmt.Errorf("failed to load vlan-ensure result: %s", err)
	}

	inf.Source = InterfaceDeviceSourceBridge{
		Bridge: bridge,
	}

	domain.Devices.Devices = append(domain.Devices.Devices, inf)
	return nil
}

func (m *kvmManager) setDefaultNetwork(args *CreateParams, seq uint16, domain *Domain) error {
	_, err := netlink.LinkByName(DefaultBridgeName)
	if err != nil {
		return fmt.Errorf("bridge '%s' not found", DefaultBridgeName)
	}

	//attach to default bridge.
	domain.Devices.Devices = append(domain.Devices.Devices, InterfaceDevice{
		Type: InterfaceDeviceTypeBridge,
		Source: InterfaceDeviceSourceBridge{
			Bridge: DefaultBridgeName,
		},
		Mac: &InterfaceDeviceMac{
			Address: m.macAddr(seq),
		},
		Model: InterfaceDeviceModel{
			Type: "virtio",
		},
	})

	if err := m.setDHCPHost(seq); err != nil {
		return err
	}

	//start port forwarders
	if err := m.setPortForwards(domain.UUID, seq, args.Port); err != nil {
		return err
	}

	return nil
}

func (m *kvmManager) setDHCPHost(seq uint16) error {
	mac := m.macAddr(seq)
	ip := m.ipAddr(seq)

	runner, err := pm.GetManager().RunCmd(&core.Command{
		ID:      uuid.New(),
		Command: "bridge.add_host",
		Arguments: core.MustArguments(map[string]interface{}{
			"bridge": DefaultBridgeName,
			"mac":    mac,
			"ip":     ip,
		}),
	})

	if err != nil {
		return err
	}
	result := runner.Wait()

	if result.State != core.StateSuccess {
		return fmt.Errorf("failed to add host to dnsmasq: %s", result.Data)
	}

	return nil
}

func (m *kvmManager) forwardId(uuid string, host int) string {
	return fmt.Sprintf("kvm-socat-%v-%v", uuid, host)
}

func (m *kvmManager) unPortForward(uuid string) {
	for key, runner := range pm.GetManager().Runners() {
		if strings.HasPrefix(key, fmt.Sprintf("kvm-socat-%s", uuid)) {
			runner.Terminate()
		}
	}
}
