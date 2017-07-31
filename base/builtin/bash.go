package builtin

import "github.com/zero-os/0-core/base/pm"

func init() {
	pm.RegisterExtension("bash", "sh", "", []string{"-c", "{script}"}, nil)
}
