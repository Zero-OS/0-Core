package builtin

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseMountCmd(t *testing.T) {
	mount := `/dev/sda2 /var/lib/docker/plugins ext4 rw,relatime,errors=remount-ro,data=ordered 0 0
/dev/sda2 /var/lib/docker/aufs ext4 rw,relatime,errors=remount-ro,data=ordered 0 0
	`
	mounts := parseMountCmd(mount)
	mountpoints, exists := mounts["/dev/sda2"]
	if ok := assert.Equal(t, true, exists); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, 2, len(mountpoints)); !ok {
		t.Fatal()
	}

	pointOne := mountpoints[0]
	if ok := assert.Equal(t, "/var/lib/docker/plugins", pointOne.Mountpoint); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "ext4", pointOne.Filesystem); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "1", pointOne.Options["rw"]); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "ordered", pointOne.Options["data"]); !ok {
		t.Fatal()
	}

	pointTwo := mountpoints[1]
	if ok := assert.Equal(t, "/var/lib/docker/aufs", pointTwo.Mountpoint); !ok {
		t.Fatal()
	}

}

func testParseSmartctlInfo(t *testing.T) {
	input := `smartctl 6.5 2016-01-24 r4214 [x86_64-linux-4.4.0-116-generic] (local build)
Copyright (C) 2002-16, Bruce Allen, Christian Franke, www.smartmontools.org

=== START OF INFORMATION SECTION ===
Device Model:     KINGSTON SHFS37A240G
Serial Number:    50026B7258099A16
LU WWN Device Id: 5 0026b7 258099a16
Firmware Version: 603ABBF0
User Capacity:    240,057,409,536 bytes [240 GB]
Sector Size:      512 bytes logical/physical
Rotation Rate:    Solid State Device
Device is:        Not in smartctl database [for details use: -P showall]
ATA Version is:   ATA8-ACS, ACS-2 T13/2015-D revision 3
SATA Version is:  SATA 3.0, 6.0 Gb/s (current: 6.0 Gb/s)
Local Time is:    Sun Apr 15 15:33:43 2018 EET
SMART support is: Available - device has SMART capability.
SMART support is: Enabled

`
	info, _ := parseSmartctlInfo(input)

	if ok := assert.Equal(t, "KINGSTON SHFS37A240G", info.Model); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "50026B7258099A16", info.SerialNumber); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "5 0026b7 258099a16", info.DeviceID); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "603ABBF0", info.FirmwareVersion); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, 240057409536, info.UserCapacity); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, 512, info.SectorSize); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "Solid State Device", info.RotationRate); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "Not in smartctl database [for details use: -P showall]", info.Device); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "ATA8-ACS, ACS-2 T13/2015-D revision 3", info.ATAVersion); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "SATA 3.0, 6.0 Gb/s (current: 6.0 Gb/s)", info.SATAVersion); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, true, info.SmartSupportAvailable); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, true, info.SmartSupportEnabled); !ok {
		t.Fatal()
	}

}
