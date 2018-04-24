package nft

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	sample = `table ip nat {
	chain pre {
		type nat hook prerouting priority 0; policy accept;
		iif "core0" mark set 0x00000001 # handle 3
		iif "kvm0" mark set 0x00000001 # handle 6
	}

	chain post {
		type nat hook postrouting priority 0; policy accept;
		ip saddr 172.18.0.0/16 masquerade # handle 5
		ip saddr 172.19.0.0/16 masquerade # handle 7
	}
}
table inet filter {
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

	chain input {
		type filter hook input priority 0; policy drop;
		ct state { established, related} accept # handle 4
		iifname "lo" accept # handle 5
		iifname "vxbackend" accept # handle 6
		ip protocol 1 accept # handle 7
		iif "core0" udp dport { bootpc, bootps, domain} accept # handle 8
		tcp dport ssh accept # handle 11
		tcp dport 6379 accept # handle 12
		iif "kvm0" udp dport { bootpc, bootps, domain} accept # handle 13
	}
}
	`
)

func TestNFTParse(t *testing.T) {
	nft, err := Parse(sample)

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Len(t, nft, 2); !ok { // 2 tables
		t.Error()
	}

	filter := nft["filter"]

	if ok := assert.Equal(t, FamilyINET, filter.Family); !ok {
		t.Error()
	}

	if ok := assert.Len(t, filter.Chains, 3); !ok { // 3 chains
		t.Error()
	}

	input := filter.Chains["input"]

	if ok := assert.Len(t, input.Rules, 8); !ok { // 7 rules
		t.Error()
	}

	if ok := assert.Equal(t, Type("filter"), input.Type); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, "input", input.Hook); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, "drop", input.Policy); !ok {
		t.Error()
	}

	rule := input.Rules[0]
	if ok := assert.Equal(t, 4, rule.Handle); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "ct state { established, related} accept", rule.Body); !ok {
		t.Fatal()
	}
}
