package main

import "io"

type ContigLines struct {
	lines     []string
	minOffset int
	maxOffset int
}

func LoadFwd(content Content, offset int, count int) (ContigLines, error) {
	r := NewForwardLineReader(content, offset)
	lines, err := load(count, r)
	if err != nil {
		return ContigLines{}, err
	}
	cl := ContigLines{lines: lines, minOffset: offset, maxOffset: offset}
	for _, l := range lines {
		cl.maxOffset += len(l)
	}
	return cl, nil
}

func LoadBck(content Content, offset int, count int) (ContigLines, error) {
	r := NewBackwardLineReader(content, offset)
	lines, err := load(count, r)
	if err != nil {
		return ContigLines{}, err
	}
	cl := ContigLines{lines: lines, minOffset: offset, maxOffset: offset}
	for _, l := range lines {
		cl.minOffset -= len(l)
	}
	return cl, nil
}

func load(count int, r LineReader) ([]string, error) {
	lines := make([]string, 0, count)
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
