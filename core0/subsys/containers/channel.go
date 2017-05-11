package containers

import (
	"io"
	"os"
	"syscall"
)

type Channel interface {
	Writer() uintptr
	Reader() uintptr
	io.ReadWriteCloser
}

type channel struct {
	r *os.File
	w *os.File
}

func (c *channel) Close() error {
	e1 := c.r.Close()
	e2 := c.w.Close()
	if e1 != nil {
		return e1
	}
	if e2 != nil {
		return e2
	}
	return nil
}

func (c *channel) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c *channel) Write(p []byte) (n int, err error) {
	return c.w.Write(p)
}

func (c *channel) Writer() uintptr {
	return c.w.Fd()
}

func (c *channel) Reader() uintptr {
	return c.r.Fd()
}

func Pipe() (Channel, Channel, error) {
	lp := make([]int, 2)
	rp := make([]int, 2)

	cl := func(fds []int) {
		for _, fd := range fds {
			syscall.Close(fd)
		}
	}

	if err := syscall.Pipe(lp); err != nil {
		return nil, nil, err
	}

	if err := syscall.Pipe(rp); err != nil {
		cl(lp)
		return nil, nil, err
	}

	lc := &channel{
		r: os.NewFile(uintptr(lp[0]), "|LR"),
		w: os.NewFile(uintptr(rp[1]), "|LW"),
	}

	rc := &channel{
		r: os.NewFile(uintptr(rp[0]), "|RR"),
		w: os.NewFile(uintptr(lp[1]), "|RW"),
	}

	return lc, rc, nil
}
