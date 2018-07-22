package main

import (
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/peterstace/dauntless/term"
)

func collectInterrupt(r Reactor, a App) {
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		for _ = range ch {
			r.Enque(a.Interrupt, "interrupt")
		}
	}()
}

func CollectFileSize(r Reactor, a App, c Content) {
	go func() {
		var lastSize int64
		var sleepFor time.Duration
		for {
			size, err := c.Size()
			if err != nil {
				r.Stop(err)
				return
			}
			resized := size != lastSize
			lastSize = size

			if resized {
				r.Enque(func() { a.FileSize(int(size)) }, "content size")
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

func CollectTermSize(r Reactor, a App) {
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGWINCH)
		sigCh <- nil // Prime so that we get a term size immediately.
		var lastRows, lastCols int
		for {
			forceRefresh := false
			select {
			case <-time.After(time.Second):
			case <-sigCh:
				forceRefresh = true
			}

			rows, cols, err := term.GetSize()
			if err != nil {
				r.Stop(err)
				return
			}
			if lastRows == rows && lastCols == cols {
				continue
			}
			lastRows, lastCols = rows, cols

			r.Enque(func() { a.TermSize(rows, cols, forceRefresh) }, "term size")
		}
	}()
}

func collectInput(r Reactor, a App) {
	go func() {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			r.Stop(err)
			return
		}
		var buf []byte
		for {
			var readIn [8]byte
			n, err := tty.Read(readIn[:])
			if err != nil {
				r.Stop(err)
				return
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
							r.Enque(func() { a.KeyPress(key) }, "input")
							buf = buf[i+1:]
						}
					}
					if !foundEnd {
						break
					}
				} else {
					// Process the chars normally.
					key := Key(buf[0])
					r.Enque(func() { a.KeyPress(key) }, "input")
					buf = buf[1:]
				}
			}
		}
	}()
}

func CollectContent(r io.Reader, reac Reactor, c Content) {
	go func() {
		buf := make([]byte, 16<<10)
		var sleepFor time.Duration
		for {
			n, err := r.Read(buf)
			if err != nil && err != io.EOF {
				reac.Stop(err)
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
