package main

import (
	"io"
)

func LoadFwd(content Content, offset int, count int) ([]string, error) {
	lines := make([]string, 0, count)
	reader := NewForwardLineReader(content, offset)
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

func LoadBck(content Content, offset int, count int) ([]string, error) {
	lines := make([]string, 0, count)
	r := NewBackwardLineReader(content, offset)
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
