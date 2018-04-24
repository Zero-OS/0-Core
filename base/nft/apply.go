package nft

import (
	"fmt"
	"io/ioutil"
	"os"

	logging "github.com/op/go-logging"
	"github.com/zero-os/0-core/base/pm"
)

var (
	log = logging.MustGetLogger("nft")
)

//ApplyFromFile applies nft rules from a file
func ApplyFromFile(cfg string) error {
	_, err := pm.System("nft", "-f", cfg)
	return err
}

//Apply (merge) nft rules
func Apply(nft Nft) error {
	data, err := nft.MarshalText()
	if err != nil {
		return err
	}
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.RemoveAll(f.Name())
	}()

	if _, err := f.Write(data); err != nil {
		return err
	}
	f.Close()

	return ApplyFromFile(f.Name())
}

//findRules validate that the sub is part of the ruleset, and fill the
//rules handle with the values from the ruleset
func findRules(ruleset, sub Nft) (Nft, error) {
	for tn, t := range sub {
		currenttable, ok := ruleset[tn]
		if !ok {
			return nil, fmt.Errorf("table %s not found", tn)
		}

		for cn, c := range t.Chains {
			currentchain, ok := currenttable.Chains[cn]
			if !ok {
				return nil, fmt.Errorf("chain %s not found in table %s", cn, tn)
			}
			for r := range c.Rules {
				for _, rr := range currentchain.Rules {
					if rr.Body == c.Rules[r].Body {
						c.Rules[r].Handle = rr.Handle
						break
					}
				}
				if c.Rules[r].Handle == 0 {
					return nil, fmt.Errorf("rule '%s' not found", c.Rules[r].Body)
				}
			}
		}
	}

	return sub, nil
}

//DropRules removes nft rules from a file
func DropRules(sub Nft) error {
	ruleset, err := Get()

	if err != nil {
		return err
	}

	sub, err = findRules(ruleset, sub)
	if err != nil {
		return err
	}
	// Two loops to achieve all or nothing
	for tn, t := range sub {
		for cn, c := range t.Chains {
			for _, r := range c.Rules {
				if err := Drop(t.Family, tn, cn, r.Handle); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

//Drop drops a single rule given a handle
func Drop(family Family, table, chain string, handle int) error {
	_, err := pm.System("nft", "delete", "rule", string(family), table, chain, "handle", fmt.Sprint(handle))
	return err
}
