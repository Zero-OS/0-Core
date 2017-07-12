package stream

import (
	"bytes"
	"container/list"
	"fmt"
)

type Buffer struct {
	*list.List
	size int
}

func NewBuffer(size int) *Buffer {
	return &Buffer{
		List: list.New(),
		size: size,
	}
}

func (b *Buffer) String() string {
	var strbuf bytes.Buffer
	for l := b.Front(); l != nil; l = l.Next() {
		switch v := l.Value.(type) {
		case string:
			strbuf.WriteString(v)
		default:
			strbuf.WriteString(fmt.Sprintf("%v", l))
		}
		strbuf.WriteString("\n")
	}

	return strbuf.String()
}

func (b *Buffer) Append(o interface{}) {
	b.PushBack(o)
	if b.Len() > b.size {
		b.Remove(b.Front())
	}
}
