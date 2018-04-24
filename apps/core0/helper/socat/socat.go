package socat

import (
	"fmt"
	"strings"
	"sync"

	"github.com/zero-os/0-core/base/nft"

	"github.com/op/go-logging"
)

var (
	log  = logging.MustGetLogger("socat")
	lock sync.Mutex

	rules = map[int]rule{}
)

type rule struct {
	ns   string
	port int
	ip   string
}

func (r rule) Rule(host int) string {
	return fmt.Sprintf("tcp dport %d dnat to %s:%d", host, r.ip, r.port)
}

//SetPortForward create a single port forward from host, to dest in this namespace
//The namespace is used to group port forward rules so they all can get terminated
//with one call later.
func SetPortForward(namespace string, ip string, host int, dest int) error {
	lock.Lock()
	defer lock.Unlock()

	//NOTE: this will only check if the port is used for port forwarding
	//if a port on the host is using this port it will get masked out
	if _, exists := rules[host]; exists {
		return fmt.Errorf("port already in use")
	}

	r := rule{
		ns:   forwardId(namespace, host, dest),
		port: dest,
		ip:   ip,
	}

	set := nft.Nft{
		"nat": nft.Table{
			Family: nft.FamilyIP,
			Chains: nft.Chains{
				"pre": nft.Chain{
					Rules: []nft.Rule{
						{Body: r.Rule(host)},
					},
				},
			},
		},
	}

	if err := nft.Apply(set); err != nil {
		return err
	}

	rules[host] = r
	return nil
}

func forwardId(namespace string, host int, dest int) string {
	return fmt.Sprintf("socat-%v-%v-%v", namespace, host, dest)
}

//RemovePortForward removes a single port forward
func RemovePortForward(namespace string, host int, dest int) error {
	lock.Lock()
	defer lock.Unlock()
	rule, ok := rules[host]
	if !ok {
		return fmt.Errorf("no port forwrard from host port: %d", host)
	}

	if rule.ns != forwardId(namespace, host, dest) {
		return fmt.Errorf("permission denied")
	}

	set := nft.Nft{
		"nat": nft.Table{
			Family: nft.FamilyIP,
			Chains: nft.Chains{
				"pre": nft.Chain{
					Rules: []nft.Rule{
						{Body: rule.Rule(host)},
					},
				},
			},
		},
	}

	return nft.Apply(set)
}

//RemoveAll remove all port forwrards that were created in this namespace.
func RemoveAll(namespace string) error {
	lock.Lock()
	defer lock.Unlock()

	var todelete []nft.Rule
	var hostPorts []int

	for host, r := range rules {
		if !strings.HasPrefix(r.ns, fmt.Sprintf("socat-%s", namespace)) {
			continue
		}

		todelete = append(todelete, nft.Rule{
			Body: r.Rule(host),
		})

		hostPorts = append(hostPorts, host)
	}

	if len(todelete) == 0 {
		return nil
	}

	set := nft.Nft{
		"nat": nft.Table{
			Family: nft.FamilyIP,
			Chains: nft.Chains{
				"pre": nft.Chain{
					Rules: todelete,
				},
			},
		},
	}

	if err := nft.DropRules(set); err != nil {
		log.Errorf("failed to delete ruleset: %s", err)
		return err
	}

	for _, host := range hostPorts {
		delete(rules, host)
	}

	return nil
}
