package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func collectTermSize(r Reactor, a App) {
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGWINCH)
		sigCh <- nil // Prime so that we get a term size immediately.
		for {

			force := false
			select {
			case <-time.After(time.Second):
			case <-sigCh:
				force = true
			}

			rows, cols, err := getTermSize()
			if err != nil {
				r.Stop(err)
				return
			}

			r.Enque(func() {
				if force {
					a.ForceRefresh()
				}
				a.TermSize(rows, cols)
			})
		}
	}()
}

func getTermSize() (rows int, cols int, err error) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	var dim []byte
	dim, err = cmd.Output()
	if err == nil {
		_, err = fmt.Sscanf(string(dim), "%d %d", &rows, &cols)
	}
	return
}
