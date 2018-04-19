package screen

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gizak/termui"

	"github.com/op/go-logging"
)

const (
	DefaultWidth  = 80
	DefaultHeight = 25

	wipeSequence      = "\033[2J\033[;H"
	resetSequence     = "\033[0;0f"
	clearLineSequence = "\033[K"
	lineFmt           = "%-80s\n"
)

var (
	log = logging.MustGetLogger("screen")

	width  int = DefaultWidth
	height int = DefaultHeight
	o      sync.Once
	tty    *os.File
	serr   error

	m  sync.RWMutex
	fb bytes.Buffer
)

func getSize(tty string) {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("ttysize < %s", tty))
	out, err := cmd.Output()
	if err != nil {
		return
	}
	fmt.Sscanf(string(out), "%d %d", &width, &height)
}

func test() {
	rows1 := [][]string{
		[]string{"header1", "header2", "header3"},
		[]string{"你好吗", "Go-lang is so cool", "Im working on Ruby"},
		[]string{"2016", "10", "11"},
	}

	table1 := termui.NewTable()
	table1.Rows = rows1
	table1.FgColor = termui.ColorWhite
	table1.BgColor = termui.ColorDefault
	table1.Y = 0
	table1.X = 0
	table1.Width = 62
	table1.Height = 7

	termui.Render(table1)

	rows2 := [][]string{
		[]string{"header1", "header2", "header3"},
		[]string{"Foundations", "Go-lang is so cool", "Im working on Ruby"},
		[]string{"2016", "11", "11"},
	}

	table2 := termui.NewTable()
	table2.Rows = rows2
	table2.FgColor = termui.ColorWhite
	table2.BgColor = termui.ColorDefault
	table2.TextAlign = termui.AlignCenter
	table2.Separator = false
	table2.Analysis()
	table2.SetSize()
	table2.BgColors[2] = termui.ColorRed
	table2.Y = 10
	table2.X = 0
	table2.Border = true

	termui.Render(table2)
}
func newScreen(vt int) error {
	if err := termui.Init(vt); err != nil {
		return err
	}
	test()
	go termui.Loop()
	// o.Do(func() {
	// 	cmd := exec.Command("chvt", fmt.Sprintf("%d", vt))
	// 	serr = cmd.Run()
	// 	if serr != nil {
	// 		return
	// 	}
	// 	ttyPath := fmt.Sprintf("/dev/tty%d", vt)
	// 	getSize(ttyPath)
	// 	tty, serr = os.OpenFile(ttyPath, syscall.O_RDWR|syscall.O_NOCTTY, 0644)
	// 	if serr == nil {
	// 		go render()
	// 	}
	// })

	return serr
}

func New(vt int) error {
	return newScreen(vt)
}

//makes sure that screen always have what in the current frame
func render() {
	fmt.Fprint(tty, wipeSequence)
	//get size
	space := make([]byte, width)
	for i := range space {
		space[i] = ' '
	}

	for {
		//tick sections
		refresh := false
		for _, section := range frame {
			if section, ok := section.(dynamic); ok {
				if section.tick() {
					refresh = true
				}
			}
		}

		if refresh {
			Refresh()
		}

		fmt.Fprint(tty, resetSequence)
		m.RLock()
		reader := bufio.NewScanner(bytes.NewReader(fb.Bytes()))
		var c int
		for reader.Scan() {
			txt := reader.Text()
			fmt.Fprint(tty, txt, clearLineSequence, "\n")
			c++
			if c >= height {
				break
			}
		}

		m.RUnlock()
		//write to end of screen
		for ; c < height-1; c++ {
			fmt.Fprint(tty, string(space), "\n")
		}
		tty.Sync()
		<-time.After(200 * time.Millisecond)
	}
}
