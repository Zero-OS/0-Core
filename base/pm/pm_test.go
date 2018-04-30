package pm

import (
	"testing"

	"github.com/naoina/toml"
	"github.com/stretchr/testify/assert"
)

func TestProcessArguments(t *testing.T) {

	values := map[string]interface{}{
		"name": "Azmy",
		"age":  36,
	}

	args := map[string]interface{}{
		"intvalue": 100,
		"strvalue": "hello {name}",
		"deeplist": []string{"my", "age", "is", "{age}"},
		"deepmap": map[string]interface{}{
			"subkey": "my name is {name} and I am {age} years old",
		},
	}

	processArgs(args, values)

	if ok := assert.Equal(t, "hello Azmy", args["strvalue"]); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, []string{"my", "age", "is", "36"}, args["deeplist"]); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, map[string]interface{}{
		"subkey": "my name is Azmy and I am 36 years old",
	}, args["deepmap"]); !ok {
		t.Error()
	}
}

func TestProcessArgumentsFromToml(t *testing.T) {

	source := `
key = "my name is {name}"
list = ["hello", "{name}"]

[sub]
sub = "my age is {age}"
	`

	var args map[string]interface{}

	if err := toml.Unmarshal([]byte(source), &args); err != nil {
		t.Fatal(err)
	}

	values := map[string]interface{}{
		"name": "Azmy",
		"age":  36,
	}

	processArgs(args, values)

	if ok := assert.Equal(t, "my name is Azmy", args["key"]); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, []interface{}{"hello", "Azmy"}, args["list"]); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, map[string]interface{}{
		"sub": "my age is 36",
	}, args["sub"]); !ok {
		t.Error()
	}
}

func TestProcessArgumentsCondition(t *testing.T) {
	values := map[string]interface{}{
		"name": "Azmy",
	}

	args := map[string]interface{}{
		"name: value": "{name}",
		"age: value":  "{age}",
	}

	processArgs(args, values)

	if ok := assert.Equal(t, "Azmy", args["value"]); !ok {
		t.Error()
	}

	values = map[string]interface{}{
		"age": "36",
	}

	args = map[string]interface{}{
		"name: value": "{name}",
		"age : value": "{age}",
	}

	processArgs(args, values)

	if ok := assert.Equal(t, "36", args["value"]); !ok {
		t.Error()
	}
}
