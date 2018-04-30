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
		"name| value": "{name}",
		"age| value":  "{age}",
	}

	processArgs(args, values)

	if ok := assert.Equal(t, "Azmy", args["value"]); !ok {
		t.Error()
	}

	values = map[string]interface{}{
		"age": "36",
	}

	args = map[string]interface{}{
		"name| value": "{name}",
		"age | value": "{age}",
	}

	processArgs(args, values)

	if ok := assert.Equal(t, "36", args["value"]); !ok {
		t.Error()
	}
}

func TestProcessArgumentsFromToml2(t *testing.T) {

	source := `
root = "https://hub.gig.tech/gig-autobuilder/zero-os-0-robot-autostart-0.5.1.flist"
name = "zrobot"
privileged = false
host_network = false

[args]
[args."not(zerotier)|port"]
"6600"=6600

[args."zerotier|port"]
"zt0:6600"=6600

[[args.nics]]
	type = "default"
`

	var args map[string]interface{}

	if err := toml.Unmarshal([]byte(source), &args); err != nil {
		t.Fatal(err)
	}

	values := map[string]interface{}{
		"zerotier": "12345",
	}

	processArgs(args, values)

	expected := map[string]interface{}{
		"nics": []interface{}{
			map[string]interface{}{"type": "default"},
		},
		"port": map[string]interface{}{
			"zt0:6600": int64(6600),
		},
	}

	if ok := assert.Equal(t, expected, args["args"]); !ok {
		t.Error()
	}
}
