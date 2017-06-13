package nft

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/zero-os/0-core/base/pm"
)

func Get() (*Nft, error) {
	job, err := pm.GetManager().System("nft", "--handle", "list", "ruleset")
	if err != nil {
		return nil, err
	}
	return Parse(job.Streams.Stdout())
}

func Parse(config string) (*Nft, error) {

	level := NFT
	chainProp := false

	nft := Nft{}
	var tablename []byte
	var chainname []byte
	var table *Table
	var chain *Chain
	var rule *Rule
	var err error
	scanner := bufio.NewScanner(strings.NewReader(config))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		switch level {
		case NFT:
			tablename, table = parseTable(line)
		case TABLE:
			chainname, chain = parseChain(line)
			if chain != nil {
				chainProp = true
			}
		case CHAIN:
			if chainProp {
				if parseChainProp(chain, line) {
					chainProp = false
				}
			} else {
				rule = parseRule(line)
				if rule != nil {
					chain.Rules = append(chain.Rules, *rule)
				}
			}
		}

		if bytes.Contains(line, []byte("{")) {
			switch level {
			case NFT:
				if table == nil {
					err = fmt.Errorf("cannot parse table")
					goto err
				}
			case TABLE:
				if chain == nil {
					err = fmt.Errorf("cannot parse chain")
					goto err
				}
			}
			level += 1
		} else if bytes.Contains(line, []byte("}")) {
			level -= 1
			switch level {
			case NFT:
				nft[string(tablename)] = *table
			case TABLE:
				table.Chains[string(chainname)] = *chain
			case CHAIN:
			default:
				err = fmt.Errorf("invalid syntax")
				goto err
			}
		}
	}
	return &nft, nil
err:
	return nil, err
}

func parseTable(line []byte) ([]byte, *Table) {
	tableRegex := regexp.MustCompile("table ([a-z0-9]+) ([a-z]+)")
	match := tableRegex.FindSubmatch(line)
	if len(match) > 0 {
		return match[2], &Table{
			Family: Family(string(match[1])),
			Chains: map[string]Chain{},
		}
	} else {
		return []byte{}, nil
	}
}

func parseChain(line []byte) ([]byte, *Chain) {
	chainRegex := regexp.MustCompile("chain ([a-z]+)")
	match := chainRegex.FindSubmatch(line)
	if len(match) > 0 {
		return match[1], &Chain{
			Rules: []Rule{},
		}
	} else {
		return []byte{}, nil
	}
}

func parseChainProp(chain *Chain, line []byte) bool {
	chainPropRegex := regexp.MustCompile("type ([a-z]+) hook ([a-z]+) priority ([0-9]+); policy ([a-z]+);")
	match := chainPropRegex.FindSubmatch(line)
	if len(match) > 0 {
		var n int
		chain.Type = Type(string(match[1]))
		chain.Hook = string(match[2])
		fmt.Sscanf(string(match[3]), "%d", &n)
		chain.Priority = n
		chain.Policy = string(match[4])
		return true
	} else {
		return false
	}
}

func parseRule(line []byte) *Rule {
	ruleRegex := regexp.MustCompile("(.+) # handle ([0-9]+)")
	match := ruleRegex.FindSubmatch(line)
	if len(match) > 0 {
		var n int
		fmt.Sscanf(string(match[2]), "%d", &n)
		return &Rule{
			Body:   string(match[1]),
			Handle: n,
		}
	} else {
		return nil
	}
}
