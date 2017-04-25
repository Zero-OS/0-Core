package screen

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/op/go-logging"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const (
	Wipe    = "\033[2J\033[;H"
	Reset   = "\033[0;0f"
	Width   = 80
	Height  = 24
	LineFmt = "%-80s\n"
)

var (
	log = logging.MustGetLogger("screen")

	o    sync.Once
	tty  *os.File
	serr error

	m  sync.RWMutex
	fb bytes.Buffer
)

func newScreen(vt int) error {
	o.Do(func() {
		cmd := exec.Command("chvt", fmt.Sprintf("%d", vt))
		serr = cmd.Run()
		if serr != nil {
			return
		}
		tty, serr = os.OpenFile(fmt.Sprintf("/dev/tty%d", vt), syscall.O_WRONLY|syscall.O_NOCTTY, 0644)
	})

	return serr
}

func New(vt int) error {
	return newScreen(vt)
}

//makes sure that screen always have what in the current frame
func render() {
	fmt.Fprint(tty, Wipe)
	//get size
	space := make([]byte, Width)
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

		fmt.Fprint(tty, Reset)
		m.RLock()
		reader := bufio.NewScanner(bytes.NewReader(fb.Bytes()))
		var c int
		for reader.Scan() {
			txt := reader.Text()
			if len(txt) > Width {
				fmt.Fprint(tty, txt[:Width], "\n")
			} else {
				fmt.Fprintf(tty, LineFmt, txt)
			}
			c++
			if c >= Height {
				break
			}
		}

		m.RUnlock()
		//write to end of screen
		for ; c < Height-1; c++ {
			fmt.Fprint(tty, string(space), "\n")
		}
		tty.Sync()
		time.Sleep(500 * time.Millisecond)
	}
}

func Render() {
	go render()
}
