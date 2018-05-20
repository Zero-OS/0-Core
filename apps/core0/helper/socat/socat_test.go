package socat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSource(t *testing.T) {
	src, err := getSource("2200")
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, source{"0.0.0.0", 2200}, src); !ok {
		t.Error()
	}

	src, err = getSource("10.20.30.40:2200")
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, source{"10.20.30.40", 2200}, src); !ok {
		t.Error()
	}

	src, err = getSource("eth0:2200")
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, source{"eth0", 2200}, src); !ok {
		t.Error()
	}

	src, err = getSource("eth0:0")
	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}

	src, err = getSource("eth0")
	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}

	src, err = getSource("")
	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}

}
