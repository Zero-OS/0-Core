package settings

import (
	"fmt"

	"github.com/zero-os/0-core/base/utils"
)

//StartupCmd startup command config
type Startup struct {
	After           []string
	RunningDelay    int
	RunningMatch    string
	RecurringPeriod int
	MaxRestart      int
	Protected       bool
	Name            string
	Tags            []string
	Args            map[string]interface{}
	Condition       string

	key string
}

func (s Startup) String() string {
	return fmt.Sprintf("[%s]/{%s}", s.Key(), s.After)
}

func (s Startup) Key() string {
	return s.key
}

func (s Startup) Weight(i *IncludedSettings, chain ...string) (int64, error) {
	if utils.InString(chain, s.Key()) {
		return 0, CyclicDependency
	}

	chain = append(chain, s.Key())
	var w int64
	if len(s.After) == 0 {
		//if no After is specified, AfterBoot value is considered
		return Priority[AfterBoot], nil
	}

	for _, a := range s.After {
		if x, ok := Priority[After(a)]; ok {
			w += x
		} else if d, ok := i.Startup[a]; ok {
			dw, err := d.Weight(i, chain...)
			if err != nil {
				return 0, err
			}

			w += dw
		} else {
			return 0, fmt.Errorf("unknown dependency '%s'", a)
		}
	}

	return w, nil
}
