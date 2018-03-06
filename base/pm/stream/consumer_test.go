package stream

import (
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testReader struct {
	chunks []string
	index  int
}

func (t *testReader) Read(p []byte) (n int, err error) {
	if len(t.chunks) == t.index {
		return 0, io.EOF
	}

	s := t.chunks[t.index]
	copy(p, []byte(s))
	t.index++
	if len(t.chunks) == t.index {
		return len(s), io.EOF
	}

	return len(s), nil
}

func (_ *testReader) Close() error {
	return nil
}

func TestConsumer_OneLine(t *testing.T) {
	var message *Message
	h := func(m *Message) {
		message = m
	}

	var wg sync.WaitGroup
	wg.Add(1)
	Consume(&wg, &testReader{
		chunks: []string{"hello world\n"},
	}, 1, h)

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

	var wg sync.WaitGroup
	wg.Add(1)
	Consume(&wg, &testReader{
		chunks: []string{
			"hello world\n",
			"10::bye bye world\n",
		},
	}, 1, h)

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

	var wg sync.WaitGroup
	wg.Add(1)
	Consume(&wg, &testReader{
		chunks: []string{
			"30:::hello\nworld\n",
			":::\n",
		},
	}, 1, h)

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

func TestConsumer_Complex(t *testing.T) {
	var messages []*Message
	h := func(m *Message) {
		messages = append(messages, m)
	}

	chunk1 := `Hello world
20::this is a single line message
30:::but this is a multi line
that spans`

	chunk2 := ` two blocks of data
:::
`
	var wg sync.WaitGroup
	wg.Add(1)
	Consume(&wg, &testReader{
		chunks: []string{
			chunk1,
			chunk2,
		},
	}, 2, h)

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

func TestConsumerNoNewLine(t *testing.T) {
	var messages []*Message
	h := func(m *Message) {
		messages = append(messages, m)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	Consume(&wg, &testReader{
		chunks: []string{
			"hello world",
		},
	}, 1, h)

	wg.Wait()

	if ok := assert.Len(t, messages, 1); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, "hello world", messages[0].Message); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, uint16(1), messages[0].Meta.Level()); !ok {
		t.Fatal()
	}
}
