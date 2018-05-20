package nft

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshal(t *testing.T) {
	nft := Nft{
		"nat": Table{
			Family: FamilyIP,
			Chains: Chains{
				"post": Chain{
					Rules: []Rule{
						{Body: fmt.Sprintf("ip saddr %s masquerade", "1.1.1.1")},
					},
				},
			},
		},
	}

	data, err := nft.MarshalText()

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}
	expected := `table ip nat {
	chain post {
		ip saddr 1.1.1.1 masquerade
	}
}
`

	if ok := assert.Equal(t, expected, string(data)); !ok {
		t.Error()
	}
}

func TestFindRules(t *testing.T) {
	config := `table ip nat {
	chain pre {
		type nat hook prerouting priority 0; policy accept;
		iif "core0" mark set 0x00000001 # handle 3
		iif "kvm0" mark set 0x00000001 # handle 6
		tcp dport 6600 dnat to 172.18.0.2:6600 # handle 9
		tcp dport 2200 dnat to 172.19.0.2:22 # handle 10
	}

	chain post {
		type nat hook postrouting priority 0; policy accept;
		ip saddr 172.18.0.0/16 masquerade # handle 5
		ip saddr 172.19.0.0/16 masquerade # handle 7
	}
}
table inet filter {
	chain input {
		type filter hook input priority 0; policy drop;
		ct state { related, established} accept # handle 4
		iifname "lo" accept # handle 5
		iifname "vxbackend" accept # handle 6
		ip protocol 1 accept # handle 7
		iif "core0" udp dport { 68, 53, 67} accept # handle 8
		tcp dport 6600 accept # handle 11
		tcp dport 6379 accept # handle 12
		tcp dport 22 accept # handle 13
		iif "kvm0" udp dport { 67, 53, 68} accept # handle 16
		tcp dport 5900 accept # handle 17
	}

	chain forward {
		type filter hook forward priority 0; policy accept;
		iif "core0" oif "core0" mark set 0x00000002 # handle 9
		oif "core0" mark 0x00000001 drop # handle 10
		iif "kvm0" oif "kvm0" mark set 0x00000002 # handle 14
		oif "kvm0" mark 0x00000001 drop # handle 15
	}

	chain output {
		type filter hook output priority 0; policy accept;
	}
}
	`

	ruleset, err := Parse(config)

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	sub := Nft{
		"nat": Table{
			Family: FamilyIP,
			Chains: Chains{
				"pre": Chain{
					Rules: []Rule{
						{Body: "tcp dport 2200 dnat to 172.19.0.2:22"},
					},
				},
			},
		},
		"filter": Table{
			Family: FamilyINET,
			Chains: Chains{
				"input": Chain{
					Rules: []Rule{
						{Body: "tcp dport 5900 accept"},
					},
				},
			},
		},
	}

	sub, err = findRules(ruleset, sub)
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, 10, sub["nat"].Chains["pre"].Rules[0].Handle); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, 17, sub["filter"].Chains["input"].Rules[0].Handle); !ok {
		t.Error()
	}
}
