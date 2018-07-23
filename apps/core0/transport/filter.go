// +build !production

package transport

import "github.com/zero-os/0-core/base/pm"

func (sink *Sink) allowedCommnad(cmd *pm.Command) bool {
	return true
}
