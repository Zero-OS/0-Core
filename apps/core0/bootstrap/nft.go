package bootstrap

import (
	"io/ioutil"

	"github.com/zero-os/0-core/base/nft"
)

var (
	nftInit = nft.Nft{
		"nat": nft.Table{
			Family: nft.FamilyIP,
			Chains: nft.Chains{
				"pre": nft.Chain{
					Type:     nft.TypeNAT,
					Hook:     "prerouting",
					Priority: 0,
					Policy:   "accept",
				},
				"post": nft.Chain{
					Type:     nft.TypeNAT,
					Hook:     "postrouting",
					Priority: 0,
					Policy:   "accept",
				},
			},
		},
		"filter": nft.Table{
			Family: nft.FamilyINET,
			Chains: nft.Chains{
				"pre": nft.Chain{
					Type:     nft.TypeFilter,
					Hook:     "prerouting",
					Priority: 0,
					Policy:   "accept",
				},
				"input": nft.Chain{
					Type:     nft.TypeFilter,
					Hook:     "input",
					Priority: 0,
					Policy:   "drop",
					Rules: []nft.Rule{
						{Body: "ct state {established, related} accept"},
						{Body: "iifname lo accept"},
						{Body: "iifname vxbackend accept"},
						{Body: "ip protocol icmp accept"},
					},
				},
				"forward": nft.Chain{
					Type:     nft.TypeFilter,
					Hook:     "forward",
					Priority: 0,
					Policy:   "accept",
				},
				"output": nft.Chain{
					Type:     nft.TypeFilter,
					Hook:     "output",
					Priority: 0,
					Policy:   "accept",
				},
			},
		},
	}

	zt = nft.Nft{
		"filter": nft.Table{
			Family: nft.FamilyINET,
			Chains: nft.Chains{
				"input": nft.Chain{
					Rules: []nft.Rule{
						{Body: `iifname "zt*" tcp dport 6379 counter packets 0 bytes 0 accept`},
						{Body: `tcp dport 6379 counter packets 0 bytes 0 drop`},
					},
				},
			},
		},
	}

	pub = nft.Nft{
		"filter": nft.Table{
			Family: nft.FamilyINET,
			Chains: nft.Chains{
				"input": nft.Chain{
					Rules: []nft.Rule{
						{Body: "tcp dport 6379 accept"},
					},
				},
			},
		},
	}
)

func (b *Bootstrap) writeRules(r string) (string, error) {
	f, err := ioutil.TempFile("", "nft")
	if err != nil {
		return "", err
	}

	defer f.Close()

	f.WriteString(r)
	return f.Name(), nil
}

func (b *Bootstrap) setNFT() error {
	return nft.Apply(nftInit)
}
