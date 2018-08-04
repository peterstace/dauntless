package dauntless

import (
	"io"
	"os"
	"syscall"
	"time"
)

func (a *app) asyncCollectInterrupt(siginterrupt <-chan os.Signal) {
	for _ = range siginterrupt {
		a.reactor.Enque(a.Interrupt, "interrupt")
	}
}

func (a *app) asyncCollectFileSize() {
	go func() {
		var lastSize int64
		var sleepFor time.Duration
		for {
			size, err := a.model.content.Size()
			if err != nil {
				a.stop(err)
			}
			resized := size != lastSize
			lastSize = size

			if resized {
				a.reactor.Enque(func() { a.FileSize(int(size)) }, "content size")
			}

			if resized {
				sleepFor = 0
			} else {
				sleepFor = 2 * (sleepFor + time.Millisecond)
				const maxSleep = time.Second
				if sleepFor > maxSleep {
					sleepFor = maxSleep
				}
			}
			time.Sleep(sleepFor)
		}
	}()
}

func (a *app) asyncCollectTermSize(termSize func() (rows int, cols int, err error), stop func(error), sigwinch chan os.Signal) {
	sigwinch <- syscall.SIGWINCH // Prime so that we get a term size immediately.
	var lastRows, lastCols int
	for {
		forceRefresh := false
		select {
		case <-time.After(time.Second):
		case <-sigwinch:
			forceRefresh = true
		}

		rows, cols, err := termSize()
		if err != nil {
			stop(err)
		}
		if lastRows == rows && lastCols == cols {
			continue
		}
		lastRows, lastCols = rows, cols

		a.reactor.Enque(func() { a.TermSize(rows, cols, forceRefresh) }, "term size")
	}
}

func (a *app) asyncCollectInput(tty io.Reader, stop func(error)) {
	var buf []byte
	for {
		var readIn [8]byte
		n, err := tty.Read(readIn[:])
		if err != nil {
			stop(err)
		}
		buf = append(buf, readIn[:n]...)
		for len(buf) > 0 {
			if len(buf) == 1 && buf[0] == '\x1b' {
				// Do nothing. Wait for the next input char to decide what to do.
				break
			} else if buf[0] == '\x1b' && buf[1] == '[' {
				// Process a multi char sequence.
				foundEnd := false
				for i := 1; i < len(buf); i++ {
					if (buf[i] >= 'A' && buf[i] <= 'Z') || buf[i] == '~' {
						foundEnd = true
						key := Key(buf[:i+1])
						a.reactor.Enque(func() { a.KeyPress(key) }, "input")
						buf = buf[i+1:]
					}
				}
				if !foundEnd {
					break
				}
			} else {
				// Process the chars normally.
				key := Key(buf[0])
				a.reactor.Enque(func() { a.KeyPress(key) }, "input")
				buf = buf[1:]
			}
		}
	}
}

func CollectContent(r io.Reader, stop func(error), c Content) {
	go func() {
		buf := make([]byte, 16<<10)
		var sleepFor time.Duration
		for {
			n, err := r.Read(buf)
			if err != nil && err != io.EOF {
				stop(err)
				return
			}

			if n > 0 {
				c.Write(buf[:n])
			}

			if n == 0 {
				sleepFor = 2 * (sleepFor + time.Millisecond)
				if sleepFor > time.Second {
					sleepFor = time.Second
				}
			} else {
				sleepFor = 0
			}
			time.Sleep(sleepFor)
		}
	}()
}
