package pm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandLoad(t *testing.T) {
	args := M{
		"name": "Azmy",
		"age":  36.0,
	}
	cmd := Command{
		ID:              "id",
		Command:         "test.builtin",
		Queue:           "my-queue",
		StatsInterval:   10,
		MaxTime:         10,
		MaxRestart:      3,
		RecurringPeriod: 20,
		Stream:          true,
		LogLevels:       []int{1, 2},
		Tags:            []string{"test", "builtin"},

		Arguments: MustArguments(args),

		//Those should not be loaded
		Flags: JobFlags{
			Protected: true,
		},
	}

	data, err := json.Marshal(cmd)

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	loaded, err := LoadCmd(data)

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, cmd.ID, loaded.ID); !ok {
		t.Error()
	}
	if ok := assert.Equal(t, cmd.Command, loaded.Command); !ok {
		t.Error()
	}
	if ok := assert.Equal(t, cmd.Queue, loaded.Queue); !ok {
		t.Error()
	}
	if ok := assert.Equal(t, cmd.StatsInterval, loaded.StatsInterval); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, cmd.MaxTime, loaded.MaxTime); !ok {
		t.Error()
	}
	if ok := assert.Equal(t, cmd.MaxRestart, loaded.MaxRestart); !ok {
		t.Error()
	}
	if ok := assert.Equal(t, cmd.RecurringPeriod, loaded.RecurringPeriod); !ok {
		t.Error()
	}
	if ok := assert.Equal(t, cmd.Stream, loaded.Stream); !ok {
		t.Error()
	}
	if ok := assert.Equal(t, cmd.LogLevels, loaded.LogLevels); !ok {
		t.Error()
	}
	if ok := assert.Equal(t, cmd.Tags, loaded.Tags); !ok {
		t.Error()
	}

	var loadedData M
	if err := json.Unmarshal(*loaded.Arguments, &loadedData); err != nil {
		t.Fatal("failed to load arguments", err)
	}

	if ok := assert.Equal(t, args, loadedData); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, JobFlags{}, loaded.Flags); !ok {
		t.Error()
	}
}
