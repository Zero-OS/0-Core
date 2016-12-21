package builtin

import "github.com/g8os/core0/base/pm"

func init() {
	pm.RegisterCmd("bash", "sh", "", []string{
		"-c",
		"T=`mktemp` && cat > $T && sh $T; EXIT=$?; rm -rf $T; exit $EXIT",
	}, nil)
}
