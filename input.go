package main

import "os"

func collectInput(fn func(b byte)) {
	go func() {
		for {
			var buf [8]byte
			n, err := os.Stdin.Read(buf[:])
			if err != nil {
				// TODO: log error. Should be non-fatal.
				continue
			}
			if n == 1 {
				fn(buf[0])
			}
		}
	}()
}
