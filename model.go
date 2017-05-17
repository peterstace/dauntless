package main

import (
	"regexp"
	"time"
)

type Model struct {
	config Config

	filename string

	rows, cols int

	// Invariants:
	//  1) If fwd is populated, then offset will match the first line.
	//  2) Fwd and bck contain consecutive lines.
	offset int
	fwd    []line
	bck    []line

	fileSize int

	tmpRegex *regexp.Regexp
	regexes  []regex

	lineWrapMode bool
	xPosition    int

	msg      string
	msgSetAt time.Time

	// TODO: Too much behaviour in here, should just be model state.
	commandReader CommandReader
}

type regex struct {
	style Style
	re    *regexp.Regexp
}

type line struct {
	offset int
	data   string
}

func (l line) nextOffset() int {
	return l.offset + len(l.data)
}
