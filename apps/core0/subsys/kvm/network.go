// +build amd64

package kvm

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"strconv"
	"strings"

	"syscall"

	"github.com/libvirt/libvirt-go"
	"github.com/pborman/uuid"
	"github.com/vishvananda/netlink"
	"github.com/zero-os/0-core/base/pm"
)

const (
	OVSTag       = "ovs"
	OVSBackPlane = "backplane"
	OVSVXBackend = "vxbackend"
)

func (m *kvmManager) setVirtNetwork(network Network) error {
	conn, err := m.libvirt.getConnection()
	if err != nil {
		return err
	}
	//conn.NetworkCreateXML()
	_, err = conn.LookupNetworkByName(network.Name)
	liberr, _ := err.(libvirt.Error)

	if err != nil && liberr.Code == libvirt.ERR_NO_NETWORK {
		data, err := xml.Marshal(network)
		if err != nil {
			return err
		}
		if _, err := conn.NetworkCreateXML(string(data)); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func (m *kvmManager) setNetworking(args *NicParams, seq uint16, domain *Domain) error {
	var (
		inf *InterfaceDevice
		err error
	)

	for _, nic := range args.Nics {
		switch nic.Type {
		case "default":
			inf, err = m.prepareDefaultNetwork(domain.UUID, seq, args.Port)
		case "bridge":
			inf, err = m.prepareBridgeNetwork(&nic)
		case "vlan":
			inf, err = m.prepareVLanNetwork(&nic)
		case "vxlan":
			inf, err = m.prepareVXLanNetwork(&nic)
		default:
			err = fmt.Errorf("unsupported network mode: %s", nic.Type)
		}
		if err != nil {
			return err
		}
		domain.Devices.Devices = append(domain.Devices.Devices, inf)
	}

	return nil
}

func (m *kvmManager) prepareVLanNetwork(nic *Nic) (*InterfaceDevice, error) {
	vlanID, err := strconv.ParseInt(nic.ID, 10, 16)
	if err != nil {
		return nil, err
	}
	if vlanID < 0 || vlanID >= 4095 {
		return nil, fmt.Errorf("invalid vlan id (0-4094)")
	}

	inf := InterfaceDevice{
		Type: InterfaceDeviceTypeNetwork,
		Model: InterfaceDeviceModel{
			Type: "virtio",
		},
	}

	if nic.HWAddress != "" {
		hw, err := net.ParseMAC(nic.HWAddress)
		if err != nil {
			return nil, err
		}
		inf.Mac = &InterfaceDeviceMac{
			Address: hw.String(),
		}
	}
	//find the container with OVS tag
	ovs := m.conmgr.GetOneWithTags(OVSTag)
	if ovs == nil {
		return nil, fmt.Errorf("ovs is needed for VLAN network type")
	}

	//ensure that a bridge is available with that vlan tag.
	//we dispatch the ovs.vlan-ensure command to container.
	result, err := m.conmgr.Dispatch(ovs.ID(), &pm.Command{
		Command: "ovs.vlan-ensure",
		Arguments: pm.MustArguments(map[string]interface{}{
			"master": OVSBackPlane,
			"vlan":   vlanID,
		}),
	})

	if err != nil {
		return nil, err
	}

	if result.State != pm.StateSuccess {
		return nil, fmt.Errorf("failed to ensure vlan bridge: %v", result.Data)
	}

	var bridge string
	if err := json.Unmarshal([]byte(result.Data), &bridge); err != nil {
		return nil, fmt.Errorf("failed to load vlan-ensure result: %s", err)
	}

	net := Network{
		Name: bridge,
	}

	net.Forward.Mode = "bridge"
	net.Bridge.Name = bridge
	net.VirtualPort.Type = "openvswitch"

	if err := m.setVirtNetwork(net); err != nil {
		return nil, err
	}

	inf.Source = InterfaceDeviceSource{
		Network: bridge,
		Bridge:  bridge,
	}
	return &inf, nil
}

func (m *kvmManager) prepareVXLanNetwork(nic *Nic) (*InterfaceDevice, error) {
	vxlan, err := strconv.ParseInt(nic.ID, 10, 64)
	if err != nil {
		return nil, err
	}
	inf := InterfaceDevice{
		Type: InterfaceDeviceTypeNetwork,
		Model: InterfaceDeviceModel{
			Type: "virtio",
		},
	}
	if nic.HWAddress != "" {
		hw, err := net.ParseMAC(nic.HWAddress)
		if err != nil {
			return nil, err
		}
		inf.Mac = &InterfaceDeviceMac{
			Address: hw.String(),
		}
	}
	//find the container with OVS tag
	ovs := m.conmgr.GetOneWithTags(OVSTag)
	if ovs == nil {
		return nil, fmt.Errorf("ovs is needed for VXLAN network type")
	}

	//ensure that a bridge is available with that vlan tag.
	//we dispatch the ovs.vxlan-ensure command to container.
	result, err := m.conmgr.Dispatch(ovs.ID(), &pm.Command{
		Command: "ovs.vxlan-ensure",
		Arguments: pm.MustArguments(map[string]interface{}{
			"master": OVSVXBackend,
			"vxlan":  vxlan,
		}),
	})

	if err != nil {
		return nil, err
	}

	if result.State != pm.StateSuccess {
		return nil, fmt.Errorf("failed to ensure vlan bridge: %v", result.Data)
	}
	//brname:
	var bridge string
	if err := json.Unmarshal([]byte(result.Data), &bridge); err != nil {
		return nil, fmt.Errorf("failed to load vlan-ensure result: %s", err)
	}

	net := Network{
		Name: bridge,
	}

	net.Forward.Mode = "bridge"
	net.Bridge.Name = bridge
	net.VirtualPort.Type = "openvswitch"

	if err := m.setVirtNetwork(net); err != nil {
		return nil, err
	}

	inf.Source = InterfaceDeviceSource{
		Network: bridge,
		Bridge:  bridge,
	}
	return &inf, nil
}

func (m *kvmManager) prepareDefaultNetwork(uuid string, seq uint16, port map[int]int) (*InterfaceDevice, error) {
	_, err := netlink.LinkByName(DefaultBridgeName)
	if err != nil {
		return nil, fmt.Errorf("bridge '%s' not found", DefaultBridgeName)
	}

	//attach to default bridge.
	inf := InterfaceDevice{
		Type: InterfaceDeviceTypeBridge,
		Source: InterfaceDeviceSource{
			Bridge: DefaultBridgeName,
		},
		Mac: &InterfaceDeviceMac{
			Address: m.macAddr(seq),
		},
		Model: InterfaceDeviceModel{
			Type: "virtio",
		},
	}

	if err := m.setDHCPHost(seq); err != nil {
		return nil, err
	}

	//start port forwarders
	if err := m.setPortForwards(uuid, seq, port); err != nil {
		return nil, err
	}
	return &inf, nil
}

func (m *kvmManager) prepareBridgeNetwork(nic *Nic) (*InterfaceDevice, error) {
	_, err := netlink.LinkByName(nic.ID)
	if err != nil {
		return nil, pm.BadRequestError(fmt.Errorf("bridge '%s' not found", nic.ID))
	}

	inf := InterfaceDevice{
		Type: InterfaceDeviceTypeBridge,
		Source: InterfaceDeviceSource{
			Bridge: nic.ID,
		},
		Model: InterfaceDeviceModel{
			Type: "virtio",
		},
	}

	if nic.HWAddress != "" {
		hw, err := net.ParseMAC(nic.HWAddress)
		if err != nil {
			return nil, err
		}
		inf.Mac = &InterfaceDeviceMac{
			Address: hw.String(),
		}
	}
	return &inf, nil
}

func (m *kvmManager) setDHCPHost(seq uint16) error {
	mac := m.macAddr(seq)
	ip := m.ipAddr(seq)

	job, err := pm.Run(&pm.Command{
		ID:      uuid.New(),
		Command: "bridge.add_host",
		Arguments: pm.MustArguments(map[string]interface{}{
			"bridge": DefaultBridgeName,
			"mac":    mac,
			"ip":     ip,
		}),
	})

	if err != nil {
		return err
	}
	result := job.Wait()

	if result.State != pm.StateSuccess {
		return fmt.Errorf("failed to add host to dnsmasq: %s", result.Data)
	}

	return nil
}

func (m *kvmManager) forwardId(uuid string, host int) string {
	return fmt.Sprintf("kvm-socat-%v-%v", uuid, host)
}

func (m *kvmManager) unPortForward(uuid string) {
	for key, job := range pm.Jobs() {
		if strings.HasPrefix(key, fmt.Sprintf("kvm-socat-%s", uuid)) {
			job.Signal(syscall.SIGTERM)
		}
	}
}
