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

type Table struct {
	Name        string
	TableFamily Family
	Chains      map[string]Chain
}

type Chain struct {
	Name   string
	Type   string
	Hook   string
	Policy string
	Rules  []Rule
}

type Rule struct {
	Handle int
	Body   string
}
