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
