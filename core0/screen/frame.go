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

type TextSection struct {
	Text string
}

func (s *TextSection) write(f io.Writer) {
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
		fmt.Fprint(f, s.Text)
	} else {
		fmt.Fprint(f, s.Text, " ", string(c))
	}
}

func (s *ProgressSection) tick() bool {
	return !s.off
}

func (s *ProgressSection) Stop(off bool) {
	s.off = off
}

type GroupSection struct {
	Sections map[string]Section
}

func (s *GroupSection) write(f io.Writer) {
	idx := 0
	for _, section := range s.Sections {
		section.write(f)
		if idx != len(s.Sections)-1 {
			f.Write([]byte{'\n'})
		}
		idx += 1
	}
}

func (s *GroupSection) tick() bool {
	v := false
	for _, sub := range s.Sections {
		if sub, ok := sub.(dynamic); ok {
			if sub.tick() {
				v = true
			}
		}
	}

	return v
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
