package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func collectTermSize(r Reactor, a App) {
	go func() {
		for {

			cmd := exec.Command("stty", "size")
			cmd.Stdin = os.Stdin
			dim, err := cmd.Output()
			if err != nil {
				r.Stop(err)
				return
			}

			var rows, cols int
			_, err = fmt.Sscanf(string(dim), "%d %d", &rows, &cols)
			if err != nil {
				r.Stop(err)
				return
			}
			r.Enque(func() { a.TermSize(rows, cols) })

			time.Sleep(time.Second)
		}
	}()
}
