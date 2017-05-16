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

func newScreen(vt int) error {
	o.Do(func() {
		cmd := exec.Command("chvt", fmt.Sprintf("%d", vt))
		serr = cmd.Run()
		if serr != nil {
			return
		}
		ttyPath := fmt.Sprintf("/dev/tty%d", vt)
		getSize(ttyPath)
		tty, serr = os.OpenFile(ttyPath, syscall.O_RDWR|syscall.O_NOCTTY, 0644)
		if serr == nil {
			go render()
		}
	})

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
