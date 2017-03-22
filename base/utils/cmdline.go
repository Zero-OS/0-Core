package utils

import (
	"io/ioutil"
	"strings"
)

func parseCmdline(content string) map[string]interface{} {
	cmdline := make(map[string]interface{})
	for _, cmdarg := range strings.Fields(content) {
		keyvalue := strings.SplitN(cmdarg, "=", 2)
		if len(keyvalue) == 1 {
			cmdline[keyvalue[0]] = ""
		} else if len(keyvalue) == 2 {
			cmdline[keyvalue[0]] = keyvalue[1]
		}
	}
	return cmdline
}

//GetCmdLine Get kernel cmdline arguments
func GetCmdLine() map[string]interface{} {
	content, err := ioutil.ReadFile("/proc/cmdline")
	if err != nil {
		log.Warning("Failed to read /proc/cmdline", err)
		return make(map[string]interface{})
	}
	return parseCmdline(string(content))
}
