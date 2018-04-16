package nft

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/zero-os/0-core/base/pm"
)

//Get gets current nft ruleset
func Get() (Nft, error) {
	job, err := pm.System("nft", "--handle", "list", "ruleset", "--numeric")
	if err != nil {
		return nil, err
	}
	return Parse(job.Streams.Stdout())
}

func Parse(config string) (Nft, error) {

	level := NFT

	nft := Nft{}
	var tablename []byte
	var chainname []byte
	var table *Table
	var tmpTable *Table
	var chain *Chain
	var tmpChain *Chain
	var rule *Rule
	scanner := bufio.NewScanner(strings.NewReader(config))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		switch level {
		case NFT:
			tablename, tmpTable = parseTable(line)
			if tmpTable != nil {
				table = tmpTable
			}
		case TABLE:
			chainname, tmpChain = parseChain(line)
			if tmpChain != nil {
				chain = tmpChain
			}
		case CHAIN:
			if bytes.HasPrefix(bytes.TrimSpace(line), []byte("type ")) {
				if err := parseChainProp(chain, line); err != nil {
					return nil, err
				}
			} else {
				rule = parseRule(line)
				if rule != nil {
					chain.Rules = append(chain.Rules, *rule)
				}
			}
		}

		if bytes.Contains(line, []byte("{")) && bytes.Contains(line, []byte("}")) {
			continue
		} else if bytes.Contains(line, []byte("{")) {
			switch level {
			case NFT:
				if table == nil {
					return nil, fmt.Errorf("cannot parse table")
				}
			case TABLE:
				if chain == nil {
					return nil, fmt.Errorf("cannot parse chain")
				}
			}
			level++
		} else if bytes.Contains(line, []byte("}")) {
			level--
			switch level {
			case NFT:
				nft[string(tablename)] = *table
			case TABLE:
				table.Chains[string(chainname)] = *chain
			case CHAIN:
			default:
				return nil, fmt.Errorf("invalid syntax: unexpected level %d", level)
			}
		}
	}
	return nft, nil
}

var (
	tableRegex     = regexp.MustCompile("table ([a-z0-9]+) ([a-z]+)")
	chainRegex     = regexp.MustCompile("chain ([a-z]+)")
	chainPropRegex = regexp.MustCompile("type ([a-z]+) hook ([a-z]+) priority ([0-9]+); policy ([a-z]+);")
	ruleRegex      = regexp.MustCompile("\\s*(.+) # handle ([0-9]+)")
)

func parseTable(line []byte) ([]byte, *Table) {
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
	match := chainRegex.FindSubmatch(line)
	if len(match) > 0 {
		return match[1], &Chain{
			Rules: []Rule{},
		}
	} else {
		return []byte{}, nil
	}
}

func parseChainProp(chain *Chain, line []byte) error {
	match := chainPropRegex.FindSubmatch(line)
	if len(match) > 0 {
		var n int
		chain.Type = Type(string(match[1]))
		chain.Hook = string(match[2])
		fmt.Sscanf(string(match[3]), "%d", &n)
		chain.Priority = n
		chain.Policy = string(match[4])
		return nil
	} else {
		return fmt.Errorf("couldn't parse line: %q", line)
	}
}

func parseRule(line []byte) *Rule {
	match := ruleRegex.FindSubmatch(line)
	if len(match) > 0 {
		var n int
		fmt.Sscanf(string(match[2]), "%d", &n)
		return &Rule{
			Body:   string(bytes.TrimSpace(match[1])),
			Handle: n,
		}
	} else {
		return nil
	}
}
