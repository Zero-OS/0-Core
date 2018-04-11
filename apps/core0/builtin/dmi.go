package builtin

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/zero-os/0-core/base/pm"
)

//DMIType (allowed types 0 -> 42)
type DMIType int

// DMI represents a map of DMISectionName to DMISection parsed from dmidecode output.

/*
Property in section is in the form of key value pairs where values are optional
and may include a list of items as well.
k: [v]
	[
		item1
		item2
		...
	]
*/
type DMI = map[string]DMISection

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

func init() {
	pm.RegisterBuiltIn("core.dmidecode", dmidecodeRunAndParse)
}

func dmidecodeRunAndParse(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Types []int `json:"typenums"`
	}
	cmdbin := "dmidecode"
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	output := ""
	var cmdargs []string
	if len(args.Types) == 0 {
		result, err := pm.System(cmdbin)
		output = result.Streams.Stdout()
		if err != nil {
			return nil, err
		}
	} else {
		for _, arg := range args.Types {
			if arg < 0 && arg > 42 {
				return nil, fmt.Errorf("Invalid type number %d", arg)
			}
			cmdargs = append(cmdargs, fmt.Sprintf(" -t %d", arg))
		}
		result, err := pm.System(cmdbin, cmdargs...)

		if err != nil {
			return nil, err
		}
		output = result.Streams.Stdout()
	}
	return ParseDMI(output)

}

// DMITypeToString returns string representation of DMIType t
func DMITypeToString(t DMIType) string {
	return dmitypeToString[t]
}

// section starts with handle until it reaches 2 new lines.
func getSections(input string) []string {
	sectionParams := regexp.MustCompile("(?ms:Handle .+?\n\n)")
	return sectionParams.FindAllString(input, -1)
}

// Extract the DMI type from the handleline.
func getDMITypeFromHandleLine(line string) (DMIType, error) {
	dmitypepat := regexp.MustCompile("DMI type ([0-9]+)")
	m := dmitypepat.FindStringSubmatch(line)
	if len(m) == 2 {
		t, err := strconv.Atoi(m[1])
		return DMIType(t), err
	}
	return 0, fmt.Errorf("Couldn't find dmitype in handleline %s", line)
}

// list property spans overs multiple indented lines.
// so we check basically if the next line isn't on the same level of indentations
func isListProperty(lidx int, lines []string) bool {
	lvl := getLineLevel(lines[lidx])
	nxtline := lines[lidx+1]
	if strings.TrimSpace(nxtline) == "" {
		return false
	}
	nxtlvl := getLineLevel(lines[lidx+1])
	return nxtlvl > lvl
}

// searches where the lines dedent again indicating the end of the property.
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

// PropertyData represents a key value pair with optional list of items
type PropertyData struct {
	Val   string   `json:"value"`
	Items []string `json:"items,omitempty"`
}

// DMISection represents a complete section like BIOS or Baseboard
type DMISection struct {
	HandleLine string                  `json:"handleline"`
	Title      string                  `json:"title"`
	TypeStr    string                  `json:"typestr,omitempty"`
	Type       DMIType                 `json:"typenum"`
	Properties map[string]PropertyData `json:"properties,omitempty"`
}

func propertyFromLine(line string) (string, PropertyData, error) {
	kvpat := regexp.MustCompile("(.+?):(.*)")
	m := kvpat.FindStringSubmatch(line)
	if len(m) == 3 {
		k, v := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		return k, PropertyData{Val: v}, nil
	} else if len(m) == 2 {
		k := strings.TrimSpace(m[1])
		return k, PropertyData{Val: ""}, nil
	} else {
		return "", PropertyData{}, fmt.Errorf("Couldnt find key value pair on the line %s", line)
	}
}

// Sections are separated by new lines.
func parseDMISection(section string) DMISection {
	dmi := DMISection{}
	lines := strings.Split(section, "\n")
	dmi.HandleLine = lines[0]
	dmi.Properties = make(map[string]PropertyData)
	if t, err := getDMITypeFromHandleLine(lines[0]); err == nil {
		dmi.Type = t
		dmi.TypeStr = dmitypeToString[dmi.Type]
	}
	dmi.Title = lines[1]

	propertieslines := lines[2:]
	for i := 0; i < len(propertieslines); i++ {
		l := propertieslines[i]
		if k, p, err := propertyFromLine(l); err == nil {
			if isListProperty(i, propertieslines) {
				endidx := whereListPropertyEnds(i, propertieslines)
				subpropslines := propertieslines[i+1 : endidx]
				for _, item := range subpropslines {
					if trimmeditem := strings.TrimSpace(item); trimmeditem != "" {
						p.Items = append(p.Items, strings.TrimSpace(item))
					}
				}
				i = endidx - 1 //skip the beginning of the new property (i will increment afterwards.)
			}
			dmi.Properties[k] = p
		}
	}
	return dmi
}

// ParseDMI Parses dmidecode output into DMI structure
func ParseDMI(input string) (DMI, error) {
	dmi := make(map[string]DMISection)
	sections := getSections(input)
	if len(sections) == 0 {
		return DMI{}, fmt.Errorf("Couldn't parse valid dmi sections from input")
	}
	for _, section := range sections {
		dmisec := parseDMISection(section)
		dmi[dmisec.Title] = dmisec
	}

	return dmi, nil
}

func getLineLevel(line string) int {
	for i, c := range line {
		if !unicode.IsSpace(c) {
			return i
		}
	}
	return 0
}
