package main

import (
	"errors"
	"os"
)

func MustOpenFile(filename string) *os.File {
	f, err := os.Open(filename)
	assert(err == nil)
	return f
}

func FindStartOfLastLine(f *os.File) (int, error) {

	info, err := f.Stat()
	if err != nil {
		return 0, err
	}

	amount := 16
	for true {

		data := make([]byte, amount)
		offset := info.Size() - int64(amount)
		if offset < 0 {
			offset = 0
		}
		_, err := f.ReadAt(data, offset)
		if err != nil {
			return 0, err
		}

		lines := extractLines(data)
		if len(lines) >= 1 {
			lastLine := lines[len(lines)-1]
			return int(info.Size()) - len(lastLine), nil
		}

		if offset == 0 {
			return 0, errors.New("could not find any complete lines")
		}

		amount *= 2
	}

	assert(false)
	return 0, nil
}
