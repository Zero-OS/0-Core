package socat

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/zero-os/0-core/base/nft"

	"github.com/op/go-logging"
)

const (
	addressAll = "0.0.0.0"
)

var (
	log  = logging.MustGetLogger("socat")
	lock sync.Mutex

	rules = map[int]rule{}
)

type source struct {
	ip   string
	port int
}

func (s source) String() string {
	if addr := net.ParseIP(s.ip); addr != nil {
		if s.ip == "0.0.0.0" {
			return fmt.Sprintf("tcp dport %d", s.port)
		}
		return fmt.Sprintf("ip saddr %s tcp dport %d", s.ip, s.port)
	} else if _, _, err := net.ParseCIDR(s.ip); err == nil {
		//NETWORK
		return fmt.Sprintf("ip saddr %s tcp dport %d", s.ip, s.port)

	}
	//assume interface name
	return fmt.Sprintf("iifname \"%s\" tcp dport %d", s.ip, s.port)
}

func getSource(src string) (source, error) {
	parts := strings.SplitN(src, ":", 2)
	var r = source{
		ip: addressAll,
	}

	if _, err := fmt.Sscanf(parts[len(parts)-1], "%d", &r.port); err != nil {
		return r, err
	}

	if r.port <= 0 || r.port >= 65536 {
		return r, fmt.Errorf("invalid port number")
	}

	if len(parts) == 2 {
		r.ip = parts[0]
	}

	return r, nil
}

//ValidHost checks if the host string is valid
//Valid hosts is (port, ip:port, or device:port)
func ValidHost(host string) bool {
	_, err := getSource(host)
	return err == nil
}

type rule struct {
	ns     string
	source source
	port   int
	ip     string
}

func (r rule) Rule() string {
	return fmt.Sprintf("%s dnat to %s:%d", r.source, r.ip, r.port)
}

//SetPortForward create a single port forward from host, to dest in this namespace
//The namespace is used to group port forward rules so they all can get terminated
//with one call later.
func SetPortForward(namespace string, ip string, host string, dest int) error {
	lock.Lock()
	defer lock.Unlock()

	src, err := getSource(host)
	if err != nil {
		return err
	}

	//NOTE: this will only check if the port is used for port forwarding
	//if a port on the host is using this port it will get masked out
	if _, exists := rules[src.port]; exists {
		return fmt.Errorf("port already in use")
	}

	r := rule{
		ns:     forwardID(namespace, src.port, dest),
		source: src,
		port:   dest,
		ip:     ip,
	}

	set := nft.Nft{
		"nat": nft.Table{
			Family: nft.FamilyIP,
			Chains: nft.Chains{
				"pre": nft.Chain{
					Rules: []nft.Rule{
						{Body: r.Rule()},
					},
				},
			},
		},
	}

	if err := nft.Apply(set); err != nil {
		return err
	}

	rules[src.port] = r
	return nil
}

func forwardID(namespace string, host int, dest int) string {
	return fmt.Sprintf("socat-%v-%v-%v", namespace, host, dest)
}

//RemovePortForward removes a single port forward
func RemovePortForward(namespace string, host string, dest int) error {
	lock.Lock()
	defer lock.Unlock()
	src, err := getSource(host)
	if err != nil {
		return err
	}

	rule, ok := rules[src.port]
	if !ok {
		return fmt.Errorf("no port forwrard from host port: %d", src.port)
	}

	if rule.ns != forwardID(namespace, src.port, dest) {
		return fmt.Errorf("permission denied")
	}

	set := nft.Nft{
		"nat": nft.Table{
			Family: nft.FamilyIP,
			Chains: nft.Chains{
				"pre": nft.Chain{
					Rules: []nft.Rule{
						{Body: rule.Rule()},
					},
				},
			},
		},
	}

	if err := nft.DropRules(set); err != nil {
		return err
	}

	delete(rules, src.port)
	return nil
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
			Body: r.Rule(),
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
