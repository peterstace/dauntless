package main

import (
	"os"
	"os/signal"
)

func collectInterrupt(r Reactor, a App) {
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		for _ = range ch {
			r.Enque(a.Interrupt)
		}
	}()
}
