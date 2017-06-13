package nft

import (
	"bytes"
	"fmt"
)

type Family string
type Type string

const (
	NFT = iota
	TABLE
	CHAIN

	FamilyIP     = Family("ip")
	FamilyIP6    = Family("ip6")
	FamilyNET    = Family("net")
	FamilyARP    = Family("arp")
	FamilyBridge = Family("bridge")

	TypeSkipCreate = Type("")
	TypeNAT        = Type("nat")
	TypeFilter     = Type("filter")
)

type Nft map[string]Table

func (n Nft) MarshalText() ([]byte, error) {
	var buf bytes.Buffer

	for name, table := range n {
		if name == "" {
			return nil, fmt.Errorf("table name is required")
		}

		if err := table.marshal(name, &buf); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

type Chains map[string]Chain

type Table struct {
	Family Family
	Chains Chains
}

func (t *Table) marshal(name string, buf *bytes.Buffer) error {
	if t.Family == Family("") {
		return fmt.Errorf("family is required")
	}
	buf.WriteString(fmt.Sprintf("table %s %s {\n", t.Family, name))
	for name, chain := range t.Chains {
		if name == "" {
			return fmt.Errorf("empty chain name")
		}

		if err := chain.marshal(name, buf); err != nil {
			return err
		}
	}

	buf.WriteString("}\n")
	return nil
}

type Chain struct {
	Type     Type
	Hook     string
	Priority int
	Policy   string
	Rules    []Rule
}

func (c *Chain) marshal(name string, buf *bytes.Buffer) error {
	//TODO: validate type and hook
	buf.WriteString(fmt.Sprintf("\tchain %s {\n", name))
	if c.Type != TypeSkipCreate {
		buf.WriteString(fmt.Sprintf("\t\ttype %s hook %s priority %d; policy %s;\n", c.Type, c.Hook, c.Priority, c.Policy))
	}

	for _, rule := range c.Rules {
		buf.WriteString("\t\t")
		buf.WriteString(rule.Body)
		buf.WriteByte('\n')
	}

	buf.WriteString("\t}\n")

	return nil
}

type Rule struct {
	Handle int
	Body   string
}
