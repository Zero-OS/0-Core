package stream

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewMeta(t *testing.T) {
	m := NewMeta(10, Flag(1), Flag(2))

	if ok := assert.Equal(t, uint16(10), m.Level()); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, m.Is(Flag(1))); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, m.Is(Flag(2))); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, m.Is(Flag(3))); !ok {
		t.Fatal()
	}

	if ok := assert.False(t, m.Is(Flag(4))); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, m.Assert(10)); !ok {
		t.Fatal()
	}

	if ok := assert.False(t, m.Assert(20)); !ok {
		t.Fatal()
	}
}

func TestNewMetaWithCode(t *testing.T) {
	m := NewMetaWithCode(100, 10, Flag(1), Flag(2))

	if ok := assert.Equal(t, uint16(10), m.Level()); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, m.Is(Flag(1))); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, m.Is(Flag(2))); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, m.Is(Flag(3))); !ok {
		t.Fatal()
	}

	if ok := assert.False(t, m.Is(Flag(4))); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, m.Assert(10)); !ok {
		t.Fatal()
	}

	if ok := assert.False(t, m.Assert(20)); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, uint32(100), m.Code()); !ok {
		t.Fatal()
	}
}

func TestMeta_Set(t *testing.T) {
	m := NewMeta(10, Flag(1))

	if ok := assert.Equal(t, uint16(10), m.Level()); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, m.Is(Flag(1))); !ok {
		t.Fatal()
	}

	if ok := assert.False(t, m.Is(Flag(2))); !ok {
		t.Fatal()
	}

	m = m.Set(Flag(2))

	if ok := assert.True(t, m.Is(Flag(2))); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, m.Is(Flag(3))); !ok {
		t.Fatal()
	}

	if ok := assert.False(t, m.Is(Flag(4))); !ok {
		t.Fatal()
	}
}

func TestNewMeta_Base(t *testing.T) {
	m := NewMetaWithCode(100, 10, Flag(1), Flag(2))
	m = m.Base()

	if ok := assert.Equal(t, uint32(0), m.Code()); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, uint16(10), m.Level()); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, m.Is(Flag(1))); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, m.Is(Flag(2))); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, m.Is(Flag(3))); !ok {
		t.Fatal()
	}

	if ok := assert.False(t, m.Is(Flag(4))); !ok {
		t.Fatal()
	}
}
