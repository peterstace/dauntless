package main

import (
	"fmt"
	"os"
)

// TODO: Add FG/BG colours.

func WriteToTerm(cells []byte) {

	for _, b := range cells {
		if b < 32 || b > 126 {
			panic(fmt.Sprintf("byte out of range [32, 126]: %d", b))
		}
	}

	os.Stdout.WriteString("\x1b[2J\x1b[H") // Clear screen and move cursor to top left.
	os.Stdout.Write(cells)
}
