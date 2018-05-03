package stream

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuffer(t *testing.T) {
	buf := NewBuffer(10)

	if ok := assert.NotNil(t, buf); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, 0, buf.Len()); !ok {
		t.Error()
	}
}

func TestBufferLimitedSize(t *testing.T) {
	size := 10
	buf := NewBuffer(size)

	if ok := assert.NotNil(t, buf); !ok {
		t.Fatal()
	}

	for i := 0; i < 20; i++ {
		buf.Append(i)
	}

	if ok := assert.Equal(t, size, buf.Len()); !ok {
		t.Error()
	}
}

func TestBufferString(t *testing.T) {
	size := 10
	buf := NewBuffer(size)

	if ok := assert.NotNil(t, buf); !ok {
		t.Fatal()
	}

	for i := 0; i < 20; i++ {
		buf.Append(fmt.Sprintf("line %d", i))
	}

	if ok := assert.Equal(t, size, buf.Len()); !ok {
		t.Error()
	}

	expected := ""
	for i := 10; i < 20; i++ {
		expected += fmt.Sprintf("line %d\n", i)
	}

	expected = strings.TrimSpace(expected)
	if ok := assert.Equal(t, expected, buf.String()); !ok {
		t.Error()
	}
}
