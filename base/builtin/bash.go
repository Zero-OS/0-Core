package builtin

import "github.com/zero-os/0-core/base/pm"

func init() {
	pm.RegisterCmd("bash", "sh", "", []string{"-c", "{script}"}, nil)
}
