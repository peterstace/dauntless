package main

import (
	"bufio"
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

	if _, err := f.Seek(int64(offset), 0); err != nil {
		return nil, err
	}

	r := bufio.NewReader(f)
	for i := 0; i < count; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return lines, nil
			} else {
				return nil, err
			}
		}
		lines = append(lines, line)
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
