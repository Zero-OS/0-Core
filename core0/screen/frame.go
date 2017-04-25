package screen

import (
	"fmt"
	"io"
)

type Section interface {
	write(io.Writer)
}

type dynamic interface {
	tick() bool
}

type Frame []Section

var (
	frame Frame
)

type StringSection struct {
	Text string
}

func (s *StringSection) write(f io.Writer) {
	fb.WriteString(s.Text)
}

type ProgressSection struct {
	Text string
	c    byte
	off  bool
}

func (s *ProgressSection) write(f io.Writer) {
	c := s.c
	switch c {
	case '-':
		c = '\\'
	case '\\':
		c = '|'
	case '|':
		c = '/'
	case '/':
		c = '-'
	default:
		c = '-'
	}

	s.c = c
	if s.off {
		fmt.Fprint(f, s.Text, " ", "DONE")
	} else {
		fmt.Fprint(f, s.Text, " ", string(c))
	}
}

func (s *ProgressSection) tick() bool {
	return !s.off
}

func (s *ProgressSection) Progress(off bool) {
	s.off = off
}

func Refresh() {
	m.Lock()
	defer m.Unlock()
	fb.Reset()
	for _, section := range frame {
		if fb.Len() > 0 {
			fb.WriteByte('\n')
		}
		section.write(&fb)
	}
}

func Push(section Section) {
	frame = append(frame, section)
}
