package main

import (
	"os"
	"os/signal"
)

func collectSignal(r Reactor, sig os.Signal, fn func()) {
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, sig)
		for _ = range ch {
			r.Enque(fn)
		}
	}()
}
