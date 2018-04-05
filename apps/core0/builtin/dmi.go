package builtin

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

//DMIType (allowed types 0 -> 42)
type DMIType int

const (
	DMITypeBIOS DMIType = iota
	DMITypeSystem
	DMITypeBaseboard
	DMITypeChassis
	DMITypeProcessor
	DMITypeMemoryController
	DMITypeMemoryModule
	DMITypeCache
	DMITypePortConnector
	DMITypeSystemSlots
	DMITypeOnBoardDevices
	DMITypeOEMSettings
	DMITypeSystemConfigurationOptions
	DMITypeBIOSLanguage
	DMITypeGroupAssociations
	DMITypeSystemEventLog
	DMITypePhysicalMemoryArray
	DMITypeMemoryDevice
	DMIType32BitMemoryError
	DMITypeMemoryArrayMappedAddress
	DMITypeMemoryDeviceMappedAddress
	DMITypeBuiltinPointingDevice
	DMITypePortableBattery
	DMITypeSystemReset
	DMITypeHardwareSecurity
	DMITypeSystemPowerControls
	DMITypeVoltageProbe
	DMITypeCoolingDevice
	DMITypeTemperatureProbe
	DMITypeElectricalCurrentProbe
	DMITypeOutOfBandRemoteAccess
	DMITypeBootIntegrityServices
	DMITypeSystemBoot
	DMIType64BitMemoryError
	DMITypeManagementDevice
	DMITypeManagementDeviceComponent
	DMITypeManagementDeviceThresholdData
	DMITypeMemoryChannel
	DMITypeIPMIDevice
	DMITypePowerSupply
	DMITypeAdditionalInformation
	DMITypeOnboardDevicesExtendedInformation
	DMITypeManagementControllerHostInterface
)

var dmitypeToString = map[DMIType]string{
	DMITypeBIOS:                              "BIOS",
	DMITypeSystem:                            "System",
	DMITypeBaseboard:                         "Baseboard",
	DMITypeChassis:                           "Chassis",
	DMITypeProcessor:                         "Processor",
	DMITypeMemoryController:                  "MemoryController",
	DMITypeMemoryModule:                      "MemoryModule",
	DMITypeCache:                             "Cache",
	DMITypePortConnector:                     "PortConnector",
	DMITypeSystemSlots:                       "SystemSlots",
	DMITypeOnBoardDevices:                    "OnBoardDevices",
	DMITypeOEMSettings:                       "OEMSettings",
	DMITypeSystemConfigurationOptions:        "SystemConfigurationOptions",
	DMITypeBIOSLanguage:                      "BIOSLanguage",
	DMITypeGroupAssociations:                 "GroupAssociations",
	DMITypeSystemEventLog:                    "SystemEventLog",
	DMITypePhysicalMemoryArray:               "PhysicalMemoryArray",
	DMITypeMemoryDevice:                      "MemoryDevice",
	DMIType32BitMemoryError:                  "32BitMemoryError",
	DMITypeMemoryArrayMappedAddress:          "MemoryArrayMappedAddress",
	DMITypeMemoryDeviceMappedAddress:         "MemoryDeviceMappedAddress",
	DMITypeBuiltinPointingDevice:             "BuiltinPointingDevice",
	DMITypePortableBattery:                   "PortableBattery",
	DMITypeSystemReset:                       "SystemReset",
	DMITypeHardwareSecurity:                  "HardwareSecurity",
	DMITypeSystemPowerControls:               "SystemPowerControls",
	DMITypeVoltageProbe:                      "VoltageProbe",
	DMITypeCoolingDevice:                     "CoolingDevice",
	DMITypeTemperatureProbe:                  "TempratureProbe",
	DMITypeElectricalCurrentProbe:            "ElectricalCurrentProbe",
	DMITypeOutOfBandRemoteAccess:             "OutOfBandRemoteAccess",
	DMITypeBootIntegrityServices:             "BootIntegrityServices",
	DMITypeSystemBoot:                        "SystemBoot",
	DMIType64BitMemoryError:                  "64BitMemoryError",
	DMITypeManagementDevice:                  "ManagementDevice",
	DMITypeManagementDeviceComponent:         "ManagementDeviceComponent",
	DMITypeManagementDeviceThresholdData:     "ManagementThresholdData",
	DMITypeMemoryChannel:                     "MemoryChannel",
	DMITypeIPMIDevice:                        "IPMIDevice",
	DMITypePowerSupply:                       "PowerSupply",
	DMITypeAdditionalInformation:             "AdditionalInformation",
	DMITypeOnboardDevicesExtendedInformation: "OnboardDeviceExtendedInformation",
	DMITypeManagementControllerHostInterface: "ManagementControllerHostInterface",
}

// DMITypeToString returns string representation of DMIType t
func DMITypeToString(t DMIType) string {
	return dmitypeToString[t]
}

func getSections(input string) []string {
	sectionParams := regexp.MustCompile("(?ms:Handle .+?\n\n)")
	return sectionParams.FindAllString(input, -1)
}
func getDMITypeFromHandleLine(line string) (DMIType, error) {
	dmitypepat := regexp.MustCompile("DMI type ([0-9]+)")
	m := dmitypepat.FindStringSubmatch(line)
	if len(m) == 2 {
		t, err := strconv.Atoi(m[1])
		return DMIType(t), err
	}
	return 0, fmt.Errorf("Couldn't find dmitype in handleline %s", line)

}

func isListProperty(lidx int, lines []string) bool {
	lvl := getLineLevel(lines[lidx])
	nxtline := lines[lidx+1]
	if strings.TrimSpace(nxtline) == "" {
		return false
	}
	nxtlvl := getLineLevel(lines[lidx+1])
	return nxtlvl != lvl
}

func whereListPropertyEnds(startIdx int, lines []string) int {
	lvl := getLineLevel(lines[startIdx])
	for i := startIdx + 1; i < len(lines); i++ {
		current := lines[i]
		if lvl == getLineLevel(current) {
			return i
		}
	}
	return len(lines)
}

// Property represents a key value pair with optional list of items
type Property struct {
	Key   string   `json:"key"`
	Val   string   `json:"value"`
	Items []string `json:"items,omitempty"`
}

// DMISection represents a complete section like BIOS or Baseboard
type DMISection struct {
	HandleLine string     `json:"handleline"`
	Title      string     `json:"title"`
	TypeStr    string     `json:"typestr,omitempty"`
	Type       DMIType    `json:"typenum"`
	Properties []Property `json:"properties,omitempty"`
}

// DMI represents a lists of DMISections parsed from dmidecode output.
type DMI struct {
	Sections []DMISection `json:"sections"`
}

func propertyFromLine(line string) (Property, error) {
	kvpat := regexp.MustCompile("(.+?):(.*)")
	m := kvpat.FindStringSubmatch(line)
	if len(m) == 3 {
		k, v := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		return Property{Key: k, Val: v}, nil
	} else if len(m) == 2 {
		k := strings.TrimSpace(m[1])
		return Property{Key: k, Val: ""}, nil
	} else {
		return Property{}, fmt.Errorf("Couldnt find key value pair on the line %s", line)
	}
}
func parseDMISection(section string) DMISection {
	dmi := DMISection{}
	lines := strings.Split(section, "\n")
	dmi.HandleLine = lines[0]
	if t, err := getDMITypeFromHandleLine(lines[0]); err == nil {
		dmi.Type = t
		dmi.TypeStr = dmitypeToString[dmi.Type]
	}
	dmi.Title = lines[1]

	propertieslines := lines[2:]
	for i := 0; i < len(propertieslines); i++ {
		l := propertieslines[i]
		if p, err := propertyFromLine(l); err == nil {
			if isListProperty(i, propertieslines) {
				endidx := whereListPropertyEnds(i, propertieslines)
				subpropslines := propertieslines[i+1 : endidx]
				for _, item := range subpropslines {
					if trimmeditem := strings.TrimSpace(item); trimmeditem != "" {
						p.Items = append(p.Items, strings.TrimSpace(item))
					}
				}
				i = endidx //skip till the end
			}
			dmi.Properties = append(dmi.Properties, p)
		}
	}
	return dmi
}

// ParseDMI Parses dmidecode output into DMI structure
func ParseDMI(input string) DMI {
	dmi := DMI{}
	sections := getSections(input)
	for _, section := range sections {
		dmisec := parseDMISection(section)
		dmi.Sections = append(dmi.Sections, dmisec)
	}
	return dmi
}

func getLineLevel(line string) int {
	for i, c := range line {
		if !unicode.IsSpace(c) {
			return i
		}
	}
	return 0
}
