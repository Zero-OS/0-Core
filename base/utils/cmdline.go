package utils

import (
	"fmt"
	"github.com/google/shlex"
	"io/ioutil"
	"strings"
)

type KernelOptions map[string][]string

func (k KernelOptions) Is(key string) bool {
	_, ok := k[key]
	return ok
}

func (k KernelOptions) Get(key string) ([]string, bool) {
	v, ok := k[key]
	return v, ok
}

func (k KernelOptions) GetLast() map[string]interface{} {
	r := make(map[string]interface{})
	for key, values := range k {
		r[key] = values[len(values)-1]
	}

	return r
}

func (k KernelOptions) String(keys ...string) string {
	var s []string
	for _, key := range keys {
		values, ok := k[key]
		if !ok {
			continue
		}

		for _, v := range values {
			if len(v) != 0 {
				s = append(s, fmt.Sprintf("%s=%s", key, v))
			} else {
				s = append(s, key)
			}
		}
	}

	return strings.Join(s, ", ")
}

func parseKerenlOptions(content string) KernelOptions {
	options := KernelOptions{}
	cmdline, _ := shlex.Split(strings.TrimSpace(content))
	for _, option := range cmdline {
		kv := strings.SplitN(option, "=", 2)
		key := kv[0]
		value := ""
		if len(kv) == 2 {
			value = kv[1]
		}
		options[key] = append(options[key], value)
	}
	return options
}

//GetCmdLine Get kernel cmdline arguments
func GetKernelOptions() KernelOptions {
	content, err := ioutil.ReadFile("/proc/cmdline")
	if err != nil {
		log.Warning("Failed to read /proc/cmdline", err)
		return KernelOptions{}
	}

	return parseKerenlOptions(string(content))
}
