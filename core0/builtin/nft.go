package builtin

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/zero-os/0-core/base/nft"
	"github.com/zero-os/0-core/base/pm"
	"github.com/zero-os/0-core/base/pm/core"
	"github.com/zero-os/0-core/base/pm/process"
)

type nftMgr struct {
	m sync.Mutex
}

func init() {
	b := &nftMgr{}
	pm.CmdMap["nft.open_port"] = process.NewInternalProcessFactory(b.openPort)
	pm.CmdMap["nft.drop_port"] = process.NewInternalProcessFactory(b.dropPort)
}

type Port struct {
	Port      int    `json:"port"`
	Interface string `json:"interface,omitempty"`
	Subnet    string `json:"subnet,omitempty"`
}

func parsePort(cmd *core.Command) (nft.Nft, error) {
	var args Port
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}
	if args.Interface != "" && args.Subnet != "" {
		return nil, fmt.Errorf("interface and subnet are both passed")
	}
	var body string

	if args.Interface != "" {
		body = fmt.Sprintf(`iifname "%s" tcp dport %d accept`, args.Interface, args.Port)
	} else if args.Subnet != "" {
		// TODO: add checks to make sure the subnet is valid
		body = fmt.Sprintf(`ip saddr %s tcp dport %d 0 accept`, args.Subnet, args.Port)
	} else {
		body = fmt.Sprintf(`tcp dport %d accept`, args.Port)
	}
	n := nft.Nft{
		"filter": nft.Table{
			Family: nft.FamilyIP,
			Chains: nft.Chains{
				"input": nft.Chain{
					Rules: []nft.Rule{
						{Body: body},
					},
				},
			},
		},
	}
	return n, nil

}

func (b *nftMgr) openPort(cmd *core.Command) (interface{}, error) {
	n, err := parsePort(cmd)
	if err != nil {
		return nil, err
	}
	return nil, nft.Apply(n)
}

func (b *nftMgr) dropPort(cmd *core.Command) (interface{}, error) {
	n, err := parsePort(cmd)
	if err != nil {
		return nil, err
	}
	if err := nft.DropRules(n); err != nil {
		return nil, err
	}
	return nil, nil
}
