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
}

type Port struct {
	Number    int    `json:"number"`
	Interface string `json:"interface,omitempty"`
	Range     string `json:"range,omitempty"`
}

func (b *nftMgr) openPort(cmd *core.Command) (interface{}, error) {
	var args Port
	if err := json.Unmarshal(*cmd.Arguments, &args); err != nil {
		return nil, err
	}

	if args.Interface == "" && args.Range == "" {
		return nil, fmt.Errorf("interface and range are not passed")
	} else if args.Interface != "" && args.Range != "" {
		return nil, fmt.Errorf("interface and range are both passed")
	}
	var body string

	if args.Interface != "" {
		body = fmt.Sprintf(`iifname "%s" tcp dport %d counter packets 0 bytes 0 accept`, args.Interface, args.Number)
	} else {
		// TODO: add checks to make sure the range is a valid range
		body = fmt.Sprintf(`ip saddr %s tcp dport %d counter packets 0 bytes 0 accept`, args.Range, args.Number)
	}
	x := nft.Nft{
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
	return nil, nft.Apply(&x)

}
