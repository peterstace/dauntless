package main

import "os"

func collectInput(r Reactor, a App) {
	go func() {
		var buf []byte
		for {
			var readIn [8]byte
			n, err := os.Stdin.Read(readIn[:])
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
							r.Enque(func() { a.KeyPress(key) })
							buf = buf[i+1:]
						}
					}
					if !foundEnd {
						break
					}
				} else {
					// Process the chars normally.
					key := Key(buf[0])
					r.Enque(func() { a.KeyPress(key) })
					buf = buf[1:]
				}
			}
		}
	}()
}
