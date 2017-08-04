// +build !race

package stream

import (
	"bytes"
	"io/ioutil"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsumer_OneLine(t *testing.T) {
	var message *Message
	h := func(m *Message) {
		message = m
	}

	buffer := bytes.Buffer{}
	reader := ioutil.NopCloser(&buffer)
	var wg sync.WaitGroup
	wg.Add(1)
	NewConsumer(&wg, reader, 1, h)
	_, err := buffer.Write([]byte("hello world\n"))

	if ok := assert.Nil(t, err); !ok {
		t.Fatal()
	}

	reader.Close()
	wg.Wait()

	if ok := assert.NotNil(t, message); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "hello world", message.Message); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, uint16(1), message.Meta.Level()); !ok {
		t.Fatal()
	}
}

func TestConsumer_TwoLines(t *testing.T) {
	var messages []*Message
	h := func(m *Message) {
		messages = append(messages, m)
	}

	buffer := bytes.Buffer{}
	reader := ioutil.NopCloser(&buffer)
	var wg sync.WaitGroup
	wg.Add(1)
	NewConsumer(&wg, reader, 1, h)
	_, err := buffer.Write([]byte("hello world\n"))

	if ok := assert.Nil(t, err); !ok {
		t.Fatal()
	}

	_, err = buffer.Write([]byte("10::bye bye world\n"))

	if ok := assert.Nil(t, err); !ok {
		t.Fatal()
	}

	reader.Close()
	wg.Wait()

	if ok := assert.Len(t, messages, 2); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "hello world", messages[0].Message); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, uint16(1), messages[0].Meta.Level()); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, "bye bye world", messages[1].Message); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, uint16(10), messages[1].Meta.Level()); !ok {
		t.Error()
	}
}

func TestConsumer_MultiLine(t *testing.T) {
	var messages []*Message
	h := func(m *Message) {
		messages = append(messages, m)
	}

	buffer := bytes.Buffer{}
	reader := ioutil.NopCloser(&buffer)
	var wg sync.WaitGroup
	wg.Add(1)
	NewConsumer(&wg, reader, 1, h)
	_, err := buffer.Write([]byte("30:::hello\n"))

	if ok := assert.Nil(t, err); !ok {
		t.Fatal()
	}

	_, err = buffer.Write([]byte("world\n"))

	if ok := assert.Nil(t, err); !ok {
		t.Fatal()
	}

	_, err = buffer.Write([]byte(":::\n"))

	if ok := assert.Nil(t, err); !ok {
		t.Fatal()
	}

	reader.Close()
	wg.Wait()

	if ok := assert.Len(t, messages, 1); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "hello\nworld", messages[0].Message); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, uint16(30), messages[0].Meta.Level()); !ok {
		t.Error()
	}
}

func TestConsumer_MultiLineNoNewLine(t *testing.T) {
	var messages []*Message
	h := func(m *Message) {
		messages = append(messages, m)
	}

	buffer := bytes.Buffer{}
	reader := ioutil.NopCloser(&buffer)
	var wg sync.WaitGroup
	wg.Add(1)
	NewConsumer(&wg, reader, 1, h)
	_, err := buffer.Write([]byte("30:::hello\n"))

	if ok := assert.Nil(t, err); !ok {
		t.Fatal()
	}

	_, err = buffer.Write([]byte("world\n"))

	if ok := assert.Nil(t, err); !ok {
		t.Fatal()
	}

	_, err = buffer.Write([]byte(":::"))
	reader.Close()
	wg.Wait()

	if ok := assert.Nil(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Len(t, messages, 1); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "hello\nworld", messages[0].Message); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, uint16(30), messages[0].Meta.Level()); !ok {
		t.Error()
	}
}

func TestConsumer_Complex(t *testing.T) {
	var messages []*Message
	h := func(m *Message) {
		messages = append(messages, m)
	}

	buffer := bytes.Buffer{}
	reader := ioutil.NopCloser(&buffer)
	var wg sync.WaitGroup
	wg.Add(1)
	NewConsumer(&wg, reader, 2, h)
	chunk1 := `Hello world
20::this is a single line message
30:::but this is a multi line
that spans`

	chunk2 := ` two blocks of data
:::
`
	_, err := buffer.Write([]byte(chunk1))

	if ok := assert.Nil(t, err); !ok {
		t.Fatal()
	}

	_, err = buffer.Write([]byte(chunk2))

	if ok := assert.Nil(t, err); !ok {
		t.Fatal()
	}

	reader.Close()
	wg.Wait()

	if ok := assert.Len(t, messages, 3); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "Hello world", messages[0].Message); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, uint16(2), messages[0].Meta.Level()); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, "this is a single line message", messages[1].Message); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, uint16(20), messages[1].Meta.Level()); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, "but this is a multi line\nthat spans two blocks of data", messages[2].Message); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, uint16(30), messages[2].Meta.Level()); !ok {
		t.Error()
	}
}
