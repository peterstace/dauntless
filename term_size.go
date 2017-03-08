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

			var rows, cols int
			if err == nil {
				_, err = fmt.Sscanf(string(dim), "%d %d", &rows, &cols)
			}
			r.Enque(func() { a.TermSize(rows, cols, err) })

			time.Sleep(100 * time.Millisecond)
		}
	}()
}
