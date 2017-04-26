package options

import (
	"io/ioutil"
	"strings"
)

type KernelOptions struct {
	verbose bool
}

func (k *KernelOptions) Verbose() bool {
	return k.verbose
}

func kernel_init(KOptions *KernelOptions) {
	bytes, _ := ioutil.ReadFile("/proc/cmdline")
	cmdline := strings.Split(strings.Trim(string(bytes), "\n"), " ")
	for _, option := range cmdline {
		if option == "verbose" {
			KOptions.verbose = true
		}
	}
}
