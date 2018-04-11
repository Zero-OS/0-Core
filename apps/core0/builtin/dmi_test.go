package builtin

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	sample1 = `
# dmidecode 3.1
Getting SMBIOS data from sysfs.
SMBIOS 2.6 present.

Handle 0x0001, DMI type 1, 27 bytes
System Information
		Manufacturer: LENOVO
		Product Name: 20042
		Version: Lenovo G560
		Serial Number: 2677240001087
		UUID: CB3E6A50-A77B-E011-88E9-B870F4165734
		Wake-up Type: Power Switch
		SKU Number: Calpella_CRB
		Family: Intel_Mobile

	`
	sample2 = `
Getting SMBIOS data from sysfs.
SMBIOS 2.6 present.

Handle 0x0000, DMI type 0, 24 bytes
BIOS Information
		Vendor: LENOVO
		Version: 29CN40WW(V2.17)
		Release Date: 04/13/2011
		ROM Size: 2048 kB
		Characteristics:
				PCI is supported
				BIOS is upgradeable
				BIOS shadowing is allowed
				Boot from CD is supported
				Selectable boot is supported
				EDD is supported
				Japanese floppy for NEC 9800 1.2 MB is supported (int 13h)
				Japanese floppy for Toshiba 1.2 MB is supported (int 13h)
				5.25"/360 kB floppy services are supported (int 13h)
				5.25"/1.2 MB floppy services are supported (int 13h)
				3.5"/720 kB floppy services are supported (int 13h)
				3.5"/2.88 MB floppy services are supported (int 13h)
				8042 keyboard services are supported (int 9h)
				CGA/mono video services are supported (int 10h)
				ACPI is supported
				USB legacy is supported
				BIOS boot specification is supported
				Targeted content distribution is supported
		BIOS Revision: 1.40

	`
	sample3 = `
# dmidecode 3.1
Getting SMBIOS data from sysfs.
SMBIOS 2.6 present.

Handle 0x0001, DMI type 1, 27 bytes
System Information
		Manufacturer: LENOVO
		Product Name: 20042
		Version: Lenovo G560
		Serial Number: 2677240001087
		UUID: CB3E6A50-A77B-E011-88E9-B870F4165734
		Wake-up Type: Power Switch
		SKU Number: Calpella_CRB
		Family: Intel_Mobile

Handle 0x000D, DMI type 12, 5 bytes
System Configuration Options
		Option 1: String1 for Type12 Equipment Manufacturer
		Option 2: String2 for Type12 Equipment Manufacturer
		Option 3: String3 for Type12 Equipment Manufacturer
		Option 4: String4 for Type12 Equipment Manufacturer

Handle 0x000E, DMI type 15, 29 bytes
System Event Log
		Area Length: 0 bytes
		Header Start Offset: 0x0000
		Data Start Offset: 0x0000
		Access Method: General-purpose non-volatile data functions
		Access Address: 0x0000
		Status: Valid, Not Full
		Change Token: 0x12345678
		Header Format: OEM-specific
		Supported Log Type Descriptors: 3
		Descriptor 1: POST memory resize
		Data Format 1: None
		Descriptor 2: POST error
		Data Format 2: POST results bitmap
		Descriptor 3: Log area reset/cleared
		Data Format 3: None

Handle 0x0011, DMI type 32, 20 bytes
System Boot Information
		Status: No errors detected

	`
	sample4 = `
# dmidecode 3.1
Getting SMBIOS data from sysfs.
SMBIOS 2.6 present.

Handle 0x0000, DMI type 0, 24 bytes
BIOS Information
		Vendor: LENOVO
		Version: 29CN40WW(V2.17)
		Release Date: 04/13/2011
		ROM Size: 2048 kB
		Characteristics:
				PCI is supported
				BIOS is upgradeable
				BIOS shadowing is allowed
				Boot from CD is supported
				Selectable boot is supported
				EDD is supported
				Japanese floppy for NEC 9800 1.2 MB is supported (int 13h)
				Japanese floppy for Toshiba 1.2 MB is supported (int 13h)
				5.25"/360 kB floppy services are supported (int 13h)
				5.25"/1.2 MB floppy services are supported (int 13h)
				3.5"/720 kB floppy services are supported (int 13h)
				3.5"/2.88 MB floppy services are supported (int 13h)
				8042 keyboard services are supported (int 9h)
				CGA/mono video services are supported (int 10h)
				ACPI is supported
				USB legacy is supported
				BIOS boot specification is supported
				Targeted content distribution is supported
		BIOS Revision: 1.40

Handle 0x002C, DMI type 4, 42 bytes
Processor Information
		Socket Designation: CPU
		Type: Central Processor
		Family: Core 2 Duo
		Manufacturer: Intel(R) Corporation
		ID: 55 06 02 00 FF FB EB BF
		Signature: Type 0, Family 6, Model 37, Stepping 5
		Flags:
				FPU (Floating-point unit on-chip)
				VME (Virtual mode extension)
				DE (Debugging extension)
				PSE (Page size extension)
				TSC (Time stamp counter)
				MSR (Model specific registers)
				PAE (Physical address extension)
				MCE (Machine check exception)
				CX8 (CMPXCHG8 instruction supported)
				APIC (On-chip APIC hardware supported)
				SEP (Fast system call)
				MTRR (Memory type range registers)
				PGE (Page global enable)
				MCA (Machine check architecture)
				CMOV (Conditional move instruction supported)
				PAT (Page attribute table)
				PSE-36 (36-bit page size extension)
				CLFSH (CLFLUSH instruction supported)
				DS (Debug store)
				ACPI (ACPI supported)
				MMX (MMX technology supported)
				FXSR (FXSAVE and FXSTOR instructions supported)
				SSE (Streaming SIMD extensions)
				SSE2 (Streaming SIMD extensions 2)
				SS (Self-snoop)
				HTT (Multi-threading)
				TM (Thermal monitor supported)
				PBE (Pending break enabled)
		Version: Intel(R) Core(TM) i3 CPU       M 370  @ 2.40GHz
		Voltage: 0.0 V
		External Clock: 1066 MHz
		Max Speed: 2400 MHz
		Current Speed: 2399 MHz
		Status: Populated, Enabled
		Upgrade: ZIF Socket
		L1 Cache Handle: 0x0030
		L2 Cache Handle: 0x002F
		L3 Cache Handle: 0x002D
		Serial Number: Not Specified
		Asset Tag: FFFF
		Part Number: Not Specified
		Core Count: 2
		Core Enabled: 2
		Thread Count: 4
		Characteristics:
				64-bit capable
	


	`
)

func TestParseSectionSimple(t *testing.T) {
	dmi, err := ParseDMI(sample1)
	if ok := assert.Equal(t, err, nil); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, 1, len(dmi)); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, 8, len(dmi["System Information"].Properties)); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "System Information", dmi["System Information"].Title); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "2677240001087", dmi["System Information"].Properties["Serial Number"].Val); !ok {
		t.Fatal()
	}

}
func TestParseSectionWithListProperty(t *testing.T) {
	dmi, err := ParseDMI(sample2)
	if ok := assert.Equal(t, err, nil); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, 1, len(dmi)); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, 6, len(dmi["BIOS Information"].Properties)); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "BIOS Information", dmi["BIOS Information"].Title); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "1.40", dmi["BIOS Information"].Properties["BIOS Revision"].Val); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, 18, len(dmi["BIOS Information"].Properties["Characteristics"].Items)); !ok {

		t.Fatal()
	}

}

func TestParseMultipleSectionsSimple(t *testing.T) {
	dmi, err := ParseDMI(sample3)
	if ok := assert.Equal(t, err, nil); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, len(dmi), 4); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "System Information", dmi["System Information"].Title); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, 8, len(dmi["System Information"].Properties)); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "System Event Log", dmi["System Event Log"].Title); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, 15, len(dmi["System Event Log"].Properties)); !ok {
		t.Fatal()
	}
	fmt.Println("")
	if ok := assert.Equal(t, "0 bytes", dmi["System Event Log"].Properties["Area Length"].Val); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, DMITypeSystemBoot, dmi["System Boot Information"].Type); !ok {
		t.Fatal()
	}
}
func TestParseMultipleSectionsWithListProperties(t *testing.T) {
	dmi, err := ParseDMI(sample4)
	if ok := assert.Equal(t, err, nil); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, 2, len(dmi)); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "BIOS Information", dmi["BIOS Information"].Title); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, len(dmi["BIOS Information"].Properties), 6); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, 18, len(dmi["BIOS Information"].Properties["Characteristics"].Items)); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "PCI is supported", dmi["BIOS Information"].Properties["Characteristics"].Items[0]); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "LENOVO", dmi["BIOS Information"].Properties["Vendor"].Val); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "Processor Information", dmi["Processor Information"].Title); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, len(dmi["Processor Information"].Properties), 24); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "2", dmi["Processor Information"].Properties["Core Count"].Val); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, 28, len(dmi["Processor Information"].Properties["Flags"].Items)); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "FPU (Floating-point unit on-chip)", dmi["Processor Information"].Properties["Flags"].Items[0]); !ok {
		t.Fatal()
	}
}
