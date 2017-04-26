package options

import (
	"io/ioutil"
	"strings"
)

type KernelOptions map[string]string

func (k KernelOptions) Is(key string) bool {
	_, ok := k[key]
	return ok
}

func (k KernelOptions) Get(key string) (string, bool) {
	v, ok := k[key]
	return v, ok
}

func getKernelParams() KernelOptions {
	options := KernelOptions{}
	bytes, _ := ioutil.ReadFile("/proc/cmdline")
	cmdline := strings.Split(strings.Trim(string(bytes), "\n"), " ")
	for _, option := range cmdline {
		kv := strings.SplitN(option, "=", 1)
		key := kv[0]
		value := ""
		if len(kv) > 1 {
			value = strings.Join(kv[1:], "=")
		}
		options[key] = value
	}
	return options
}
