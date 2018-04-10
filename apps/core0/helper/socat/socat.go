package socat

import (
	"fmt"
	"strings"
	"sync"
	"syscall"

	"encoding/json"

	"github.com/op/go-logging"
	"github.com/zero-os/0-core/base/builtin"
	"github.com/zero-os/0-core/base/pm"
)

var (
	log  = logging.MustGetLogger("socat")
	lock sync.Mutex
)

func SetPortForward(id string, ip string, host int, dest int) error {
	lock.Lock()
	defer lock.Unlock()

	var ports []*builtin.Port
	job, err := pm.Run(&pm.Command{
		Command: "info.port",
	})
	result := job.Wait()
	if err := json.Unmarshal([]byte(result.Data), &ports); err != nil {
		pm.BadRequestError(err)
	}

	for _, port := range ports {
		if port.Network != "unix" && int(port.Port) == host {
			return fmt.Errorf("Can't forward from port %d, host is already listening on it", host)
		}
	}

	//nft add rule nat prerouting iif eth0 tcp dport { 80, 443 } dnat 192.168.1.120
	cmd := &pm.Command{
		ID:      forwardId(id, host, dest),
		Command: pm.CommandSystem,
		Flags: pm.JobFlags{
			NoOutput: true,
		},
		Arguments: pm.MustArguments(
			pm.SystemCommandArguments{
				Name: "socat",
				Args: []string{
					fmt.Sprintf("tcp-listen:%d,reuseaddr,fork", host),
					fmt.Sprintf("tcp-connect:%s:%d", ip, dest),
				},
			},
		),
	}
	onExit := &pm.ExitHook{
		Action: func(s bool) {
			log.Infof("Port forward %d:%d with id %d exited", host, dest, id)
		},
	}

	_, err = pm.Run(cmd, onExit)
	return err
}

func forwardId(id string, host int, dest int) string {
	return fmt.Sprintf("socat-%v-%v-%v", id, host, dest)
}

func RemovePortForward(id string, host int, dest int) error {
	return pm.Kill(forwardId(id, host, dest))
}

func RemoveAll(id string) {
	for key, job := range pm.Jobs() {
		if strings.HasPrefix(key, fmt.Sprintf("socat-%s", id)) {
			job.Signal(syscall.SIGTERM)
		}
	}
}
