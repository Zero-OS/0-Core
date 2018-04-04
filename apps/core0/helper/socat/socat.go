package socat

import (
	"github.com/zero-os/0-core/base/pm"
	"fmt"
	"github.com/op/go-logging"
)


var (
	log = logging.MustGetLogger("socat")
)

func SetPortForward(id string, ip string, host int, container int) error {
	//nft add rule nat prerouting iif eth0 tcp dport { 80, 443 } dnat 192.168.1.120
	cmd := &pm.Command{
		ID:      id,
		Command: pm.CommandSystem,
		Flags: pm.JobFlags{
			NoOutput: true,
		},
		Arguments: pm.MustArguments(
			pm.SystemCommandArguments{
				Name: "socat",
				Args: []string{
					fmt.Sprintf("tcp-listen:%d,reuseaddr,fork", host),
					fmt.Sprintf("tcp-connect:%s:%d", ip, container),
				},
			},
		),
	}
	onExit := &pm.ExitHook{
		Action: func(s bool) {
			log.Infof("Port forward %d:%d with id %d exited", host, container, id)
		},
	}

	_, err := pm.Run(cmd, onExit)
	return err
}
