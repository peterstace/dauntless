package main

import (
	"io"
	"os"
)

func FindJumpToBottomOffset(filename string) (int, error) {

	// TODO: This implementation is a bit silly... Would be better to just load
	// successive chunks backwards from the end of the file until we find the
	// first newline.

	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}

	info, err := f.Stat()
	if err != nil {
		return 0, err
	}

	amount := 1024
	for true {

		data := make([]byte, amount)
		offset := info.Size() - int64(amount)
		if offset < 0 {
			offset = 0
		}
		_, err := f.ReadAt(data, offset)
		if err != nil && err != io.EOF {
			return 0, err
		}

		// Throw away the first part of the data, up until the first newline.
		// This is because when we later extract the lines, it's assumed that
		// the data begins at the start of a line (which it may not).
		if startGoodData, ok := findFirstNewLine(data); ok {
			data = data[startGoodData+1:]
		} else {
			data = nil
		}

		lines := extractLines(data)
		if len(lines) >= 1 {
			startOfLine := int(info.Size())
			for i := len(lines) - 1; i >= len(lines)-1; i-- {
				startOfLine -= len(lines[i])
			}
			return startOfLine, nil
		}

		if offset == 0 {
			// Got all the back back to the start of the file, and still
			// couldn't find the required number of lines. So the required
			// position is just the start of the file.
			return 0, nil
		}

		amount *= 2
	}

	assert(false)
	return 0, nil
}
