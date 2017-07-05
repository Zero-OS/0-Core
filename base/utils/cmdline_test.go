package utils

import (
	"fmt"
	"testing"
)

func validateKeyValue(mapping KernelOptions, key string, value string, t *testing.T) {
	if val, ok := mapping[key]; ok {
		if val[0] != value {
			t.Fatal(fmt.Printf("Value %s != %s", val, value))
		}
	} else {
		t.Fatal(fmt.Printf("Key %s missing", key))
	}

}

func TestCmdParsing(t *testing.T) {
	cmdline := parseKerenlOptions("zerotier=mynetwork")
	validateKeyValue(cmdline, "zerotier", "mynetwork", t)

	cmdline = parseKerenlOptions(`something   zerotier="my network"  rgergerger`)
	validateKeyValue(cmdline, "zerotier", "my network", t)
}
