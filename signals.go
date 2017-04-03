package main

import (
	"os"
	"os/signal"
)

func collectSignals(r Reactor, a App) {
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		for sig := range ch {
			r.Enque(func() { a.Signal(sig) })
		}
	}()
}
