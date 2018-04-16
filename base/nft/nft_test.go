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
