package screen

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/op/go-logging"
	"github.com/zero-os/0-core/base/pm"
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

	path   string
	width  int = DefaultWidth
	height int = DefaultHeight
	o      sync.Once
	tty    *os.File
	serr   error

	m sync.RWMutex

	refresh chan int
)

func getSize(tty string) {
	result, err := pm.System("sh", "-c", fmt.Sprintf("ttysize < %s", tty))
	if err != nil {
		return
	}
	fmt.Sscanf(string(result.Streams.Stdout()), "%d %d", &width, &height)
}

func newScreen(vt int) error {
	o.Do(func() {
		_, serr = pm.System("chvt", fmt.Sprintf("%d", vt))
		if serr != nil {
			return
		}
		path = fmt.Sprintf("/dev/tty%d", vt)
		getSize(path)

		go render()
	})

	return serr
}

//New initialize new screen on tty2
func New(vt int) error {
	return newScreen(vt)
}

//makes sure that screen always have what in the current frame
func render() {
	//fmt.Fprint(tty, wipeSequence)
	//get size
	space := make([]byte, width)
	for i := range space {
		space[i] = ' '
	}
	refresh = make(chan int, 1)

	var fb bytes.Buffer
	for {
		fb.Reset()
		m.RLock()
		for _, section := range frame {
			if fb.Len() > 0 {
				fb.WriteByte('\n')
			}
			section.write(&fb)
		}
		m.RUnlock()

		tty, err := os.OpenFile(path, syscall.O_RDWR|syscall.O_NOCTTY, 0644)
		if err != nil {
			log.Error("failed to open screen terminal: %s", err)
			<-time.After(time.Second)
			continue
		}

		fmt.Fprint(tty, resetSequence)
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

		for ; c < height-1; c++ {
			fmt.Fprint(tty, string(space), "\n")
		}

		tty.Close()
		<-refresh
	}
}
