package utils

import (
	"fmt"
	"testing"
)

func validateKeyValue(mapping map[string]interface{}, key string, value string, t *testing.T) {
	if val, ok := mapping[key]; ok {
		if val != value {
			t.Fatal(fmt.Printf("Value %s != %s", val, value))
		}
	} else {
		t.Fatal(fmt.Printf("Key %s missing", key))
	}

}

func TestCmdParsing(t *testing.T) {
	cmdline := parseCmdline("zerotier=mynetwork")
	validateKeyValue(cmdline, "zerotier", "mynetwork", t)

	cmdline = parseCmdline("something   zerotier=mynetwork  rgergerger")
	validateKeyValue(cmdline, "zerotier", "mynetwork", t)
}
