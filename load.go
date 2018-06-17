package main

import (
	"io"
	"os"
)

func LoadFwd(filename string, offset int, count int) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lines := make([]string, 0, count)

	reader := NewForwardLineReader(f, offset)
	for i := 0; i < count; i++ {
		line, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				return lines, nil
			} else {
				return nil, err
			}
		}
		lines = append(lines, string(line))
	}
	return lines, nil
}

func LoadBck(filename string, offset int, count int) ([]string, error) {

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lines := make([]string, 0, count)

	r := NewBackwardLineReader(f, offset)
	for i := 0; i < count; i++ {
		line, err := r.ReadLine()
		if err != nil {
			if err == io.EOF {
				return lines, nil
			} else {
				return nil, err
			}
		}
		lines = append(lines, string(line))
	}

	return lines, nil
}
