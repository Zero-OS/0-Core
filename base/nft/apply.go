package nft

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/zero-os/0-core/base/pm"
)

func ApplyFromFile(cfg string) error {
	_, err := pm.GetManager().System("nft", "-f", cfg)
	return err
}

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

func DropRules(nft Nft) error {
	current, err := Get()

	if err != nil {
		return err
	}

	for tn, t := range nft {
		currenttable, ok := current[tn]
		if !ok {
			return fmt.Errorf("table %s not found", tn)
		}
		for cn, c := range t.Chains {
			currentchain, ok := currenttable.Chains[cn]
			if !ok {
				return fmt.Errorf("chain %s not found in table %s", cn, tn)
			}
			for r := range c.Rules {
				for _, rr := range currentchain.Rules {
					if rr.Body == c.Rules[r].Body {
						c.Rules[r].Handle = rr.Handle
						break
					}
				}
				if c.Rules[r].Handle == 0 {
					return fmt.Errorf("rule \"%s\" not found", c.Rules[r].Body)
				}
			}
		}
	}
	// Two loops to achieve all or nothing
	for tn, t := range nft {
		for cn, c := range t.Chains {
			for _, r := range c.Rules {
				if err := Drop(tn, cn, r.Handle); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func Drop(table, chain string, handle int) error {
	_, err := pm.GetManager().System("nft", "delete", "rule", table, chain, "handle", fmt.Sprint(handle))
	return err
}
