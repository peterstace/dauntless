package main

import "os"

func collectInput(r Reactor, a App) {
	go func() {
		for {
			// TODO: Should handle multi byte sequences.
			var buf [8]byte
			n, err := os.Stdin.Read(buf[:])
			if err != nil {
				// TODO: log error. Should be non-fatal.
				continue
			}
			if n == 1 {
				r.Enque(func() { a.KeyPress(buf[0]) })
			}
		}
	}()
}
