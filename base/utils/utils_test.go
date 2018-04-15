package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeNormalize(t *testing.T) {
	p := "../boot/kernel"

	v := SafeNormalize(p)

	if ok := assert.Equal(t, "/boot/kernel", v); !ok {
		t.Error()
	}

	p = "/boot/../kernel"
	v = SafeNormalize(p)

	if ok := assert.Equal(t, "/kernel", v); !ok {
		t.Error()
	}
}

func TestExpand(t *testing.T) {
	list, err := Expand("1,2,5-10")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, []int{1, 2, 5, 6, 7, 8, 9, 10}, list); !ok {
		t.Error()
	}
}

func TestExpandAsterisk(t *testing.T) {
	list, err := Expand("*")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, validLevels, list); !ok {
		t.Error()
	}
}

func TestExpandOutOfRange(t *testing.T) {
	list, err := Expand("50,60-100")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, []int{}, list); !ok {
		t.Error()
	}
}

func TestFormat(t *testing.T) {
	in := map[string]interface{}{
		"name": "Test",
	}

	out := Format("Hello {name}", in)

	if ok := assert.Equal(t, "Hello Test", out); !ok {
		t.Error()
	}

	out = Format("My age is {age}", in)

	if ok := assert.Equal(t, "My age is {age}", out); !ok {
		t.Error()
	}
}

func TestExists(t *testing.T) {
	name := "/etc/resolv.conf"

	if ok := assert.True(t, Exists(name)); !ok {
		t.Error()
	}

	if ok := assert.False(t, Exists("/tmp/this-file-does-not-exist-hopefully")); !ok {
		t.Error()
	}
}
