package main

import "os"

// TODO: Add FG/BG colours.

// TODO: Some kind of "delta" approach may be better. To send less to the
// screen each time.

func WriteToTerm(cells []byte) {

	for _, b := range cells {
		assert(b >= 32 && b <= 126)
	}

	os.Stdout.WriteString("\x1b[2J\x1b[H") // Clear screen and move cursor to top left.
	os.Stdout.Write(cells)
}
