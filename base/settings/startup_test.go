package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStartupNoDep(t *testing.T) {
	var startup Startup
	var include IncludedSettings

	weight, err := startup.Weight(&include)

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, Priority[AfterBoot], weight); !ok {
		t.Error()
	}
}

func TestStartupAfterMilestones(t *testing.T) {
	var startup = Startup{
		After: []string{"init"},
	}

	var include IncludedSettings

	weight, err := startup.Weight(&include)

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, Priority[AfterInit], weight); !ok {
		t.Error()
	}

	startup = Startup{
		After: []string{"net"},
	}

	weight, err = startup.Weight(&include)

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, Priority[AfterNet], weight); !ok {
		t.Error()
	}
}

func TestStartup(t *testing.T) {
	var startup = Startup{
		After: []string{"init", "custom"},
	}

	include := IncludedSettings{
		Startup: map[string]Startup{
			"custom": {
				After: []string{"init"},
			},
		},
	}

	include.prepare()

	weight, err := startup.Weight(&include)

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, Priority[AfterInit]+1, weight); !ok {
		t.Error()
	}
}
