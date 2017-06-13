package utils

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/g8os/core0/base/pm"
)

const (
	nft   = iota
	table = iota
	chain = iota
)

type NFT map[string]Table

type TableProperties struct {
}

type Table struct {
	Properties TableProperties
	Chains     map[string]Chain
}

type Chain struct {
	Type   string
	Hook   string
	Policy string
	Rules  []Rules
}

type Rule struct {
	Handle int
	Body   string
}

func main() (error, NFT) {

	job, err := pm.GetManager().System("nft", "--handle", "list", "ruleset")
	if err != nil {
		return err
	}
	level := nft
	stack := make([]interface{}, 4)
	nft := NFT{}
	var table Table
	var chain Chain
	var rule Rule
	scanner := bufio.NewScanner(strings.NewReader(job.Streams.Stdout))
	for scanner.Scan() {
		line := scanner.Bytes()

		if byte.Contains(line, []byte('{')) {
			level += 1
		} else if byte.Contains(line, []byte('}')) {
			level -= 1
		}
	}

}
