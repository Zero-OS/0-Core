package screen

import "io"

type Section interface {
	write(io.Writer)
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
