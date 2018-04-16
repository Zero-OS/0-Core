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
type DMI map[string]DMISection

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

var dmiKeywords = map[string]bool{
	"bios":      true,
	"system":    true,
	"baseboard": true,
	"chassis":   true,
	"processor": true,
	"memory":    true,
	"cache":     true,
	"connector": true,
	"slot":      true,
}

var dmiTypeRegex = regexp.MustCompile("DMI type ([0-9]+)")
var kvRegex = regexp.MustCompile("(.+?):(.*)")

func init() {
	pm.RegisterBuiltIn("core.dmidecode", dmidecodeRunAndParse)
}

func dmidecodeRunAndParse(cmd *pm.Command) (interface{}, error) {
	var args struct {
		Types []interface{} `json:"types"`
	}
	cmdbin := "dmidecode"
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	output := ""
	var cmdargs []string
	if len(args.Types) > 0 {
		for _, arg := range args.Types {
			switch iarg := arg.(type) {
			case float64:
				num := int(iarg)
				if num < 0 || num > 42 {
					return nil, pm.BadRequestError(fmt.Errorf("type out of range: %v", num))
				}
			case string:
				if exists := dmiKeywords[iarg]; !exists {
					return nil, fmt.Errorf("invalid keyword %v", arg)
				}
			default:
				return nil, pm.BadRequestError(fmt.Errorf("invalid type: %v(%T)", iarg, iarg))
			}
			cmdargs = append(cmdargs, "-t", fmt.Sprintf("%v", arg))
		}
	}

	result, err := pm.System(cmdbin, cmdargs...)

	if err != nil {
		return nil, err
	}
	output = result.Streams.Stdout()

	return ParseDMI(output)


}

// DMITypeToString returns string representation of DMIType t
func DMITypeToString(t DMIType) string {
	return dmitypeToString[t]
}

// Extract the DMI type from the handleline.
func getDMITypeFromHandleLine(line string) (DMIType, error) {
	m := dmiTypeRegex.FindStringSubmatch(line)
	if len(m) == 2 {
		t, err := strconv.Atoi(m[1])
		return DMIType(t), err
	}
	return 0, fmt.Errorf("couldn't find dmitype in handleline %s", line)
}

func getLineLevel(line string) int {
	for i, c := range line {
		if !unicode.IsSpace(c) {
			return i
		}
	}
	return 0
}

func propertyFromLine(line string) (string, PropertyData, error) {
	m := kvRegex.FindStringSubmatch(line)
	if len(m) == 3 {
		k, v := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		return k, PropertyData{Val: v}, nil
	} else if len(m) == 2 {
		k := strings.TrimSpace(m[1])
		return k, PropertyData{Val: ""}, nil
	} else {
		return "", PropertyData{}, fmt.Errorf("couldn't find key value pair on the line %s", line)
	}
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

func newSection() DMISection {
	return DMISection{
		Properties: make(map[string]PropertyData),
	}
}

func readSection(section *DMISection, lines []string, start int) (int, error){
	if (start+2) > len(lines) {
		return 0, fmt.Errorf("invalid section size")
	}

	section.HandleLine = lines[start]
	start++
	section.Title = lines[start]
	start++
	dmitype, err := getDMITypeFromHandleLine(section.HandleLine)

	if err != nil {
		return 0, err
	}

	section.Type = dmitype
	section.TypeStr = DMITypeToString(dmitype)


	for start < len(lines){
		line := lines[start]
		if strings.TrimSpace(line) == "" {
			return start, nil
		}
		indentLevel := getLineLevel(line)
		key, propertyData, err := propertyFromLine(line)
		if err != nil {
			return 0, err
		}
	    nxtIndentLevel := 0
		if len(lines) > start + 1 {
			nxtIndentLevel = getLineLevel(lines[start+1])
		}

		if nxtIndentLevel > indentLevel {
			start = readList(&propertyData, lines, start+1)
		}

		start++
		section.Properties[key] = propertyData
	}
	return start, nil
}

func readList(propertyData *PropertyData, lines []string, start int)( int){
	startIndentLevel := getLineLevel(lines[start])
	for start <len(lines) {
		line := lines[start]
		indentLevel := getLineLevel(line)
		
		if indentLevel == startIndentLevel {
			propertyData.Items = append(propertyData.Items, strings.TrimSpace(line))
		} else {
			return start - 1
		}
		start++
	}
	return start 
}


// ParseDMI Parses dmidecode output into DMI structure
func ParseDMI(input string) (DMI, error) {
	lines := strings.Split(input, "\n")
	secs := make(map[string]DMISection)

	for start := 0; start<len(lines) ; start++ {
		line := lines[start]
		if strings.HasPrefix(line, "Handle") {
			section := newSection()
			var err error
			start, err = readSection(&section, lines, start)
			if err != nil {
				return DMI{}, err
			}
			secs[section.Title] = section
		}	
	}
	return secs, nil
}