package screen

import (
	"bytes"
	"fmt"
	"io"
	"strings"
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

type Attribute string
type Attributes []Attribute

const (
	Bold  Attribute = "1"
	Red   Attribute = "31"
	Blue  Attribute = "34"
	Green Attribute = "32"
)

type TextSection struct {
	Attributes Attributes
	Text       string
}

func (s *TextSection) write(f io.Writer) {
	if len(s.Attributes) > 0 {
		fmt.Fprint(f, "\033[")
		for i, attr := range s.Attributes {
			if i > 0 {
				fmt.Fprint(f, ";")
			}
			fmt.Fprint(f, attr)
		}
		fmt.Fprint(f, "m")
	}
	fmt.Fprint(f, s.Text, "\033[0m")
}

func (s *TextSection) String() string {
	var buf bytes.Buffer
	s.write(&buf)
	return buf.String()
}

type ProgressSection struct {
	Text  string
	clock TextSection
	off   bool
}

func (s *ProgressSection) write(f io.Writer) {
	if len(s.clock.Attributes) == 0 {
		s.clock.Attributes = Attributes{Bold, Blue}
	}
	c := s.clock.Text
	switch c {
	case "-":
		c = "\\"
	case "\\":
		c = "|"
	case "|":
		c = "/"
	case "/":
		c = "-"
	default:
		c = "-"
	}

	s.clock.Text = c
	fmt.Fprint(f, s.Text)

	if !s.off {
		fmt.Fprint(f, " ")
		s.clock.write(f)
	}
}

func (s *ProgressSection) tick() bool {
	return !s.off
}

func (s *ProgressSection) Stop(off bool) {
	s.off = off
}

type GroupSection struct {
	Sections []Section
}

func (s *GroupSection) write(f io.Writer) {
	for idx, section := range s.Sections {
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

type SplitterSection struct {
	Title string
}

func (s *SplitterSection) pad(f io.Writer, padding []byte, c int) {
	for ; c > 0; c-- {
		f.Write(padding)
	}
}

func (s *SplitterSection) write(f io.Writer) {
	txt := TextSection{
		Attributes: Attributes{Bold},
		Text:       fmt.Sprintf(" %s ", s.Title),
	}
	str := txt.String()
	c := width - StringWidth(str)
	w := c / 2
	s.pad(f, []byte{'='}, w)
	fmt.Fprint(f, str)
	if 2*w < c {
		w++
	}
	s.pad(f, []byte{'='}, w)
}

type RowCell struct {
	Text string
}

type RowSection struct {
	Cells []RowCell
}

func (r *RowSection) write(f io.Writer) {
	width := width / len(r.Cells)
	s := fmt.Sprintf("%%-%ds", width)
	for _, cell := range r.Cells {
		fmt.Fprintf(f, s, cell.Text)
	}
}

type CenteredText struct {
	TextSection
}

func (c *CenteredText) write(f io.Writer) {
	str := c.String()
	w := 0
	lines := strings.Split(str, "\n")
	for _, l := range lines {
		lw := StringWidth(l)
		if lw > w {
			w = lw
		}
	}
	prefix := (width - w) / 2
	if prefix <= 0 {
		fmt.Fprint(f, str)
		return
	}

	for _, l := range lines {
		fmt.Fprintf(f, fmt.Sprintf("%%-%ds", prefix), "") //spaces
		fmt.Fprint(f, l, "\n")
	}
}

//Refresh redraws the screen after an update of the sctions
func Refresh() {
	select {
	case refresh <- 1:
	default:
	}
}

//Push section to screen
func Push(section Section) {
	m.Lock()
	frame = append(frame, section)
	m.Unlock()
}
