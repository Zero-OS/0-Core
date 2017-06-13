package nft

type Family int

const (
	NFT = iota
	TABLE
	CHAIN

	IP Family = iota
	IP6
	NET
	ARP
	BRIDGE
)

type Nft map[string]Table
type Chains map[string]Chain

type Table struct {
	Family Family
	Chains Chains
}

type Chain struct {
	Type   string
	Hook   string
	Policy string
	Rules  []Rule
}

type Rule struct {
	Handle int
	Body   string
}
