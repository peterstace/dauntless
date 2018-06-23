package main

import (
	"fmt"
	"io"
	"regexp"
)

func jumpToMatch(r Reactor, m *Model, reverse bool) {
	re := currentRE(m)
	if re == nil {
		msg := "no regex to jump to"
		log.Info(msg)
		setMessage(m, msg)
		return
	}

	if len(m.fwd) == 0 {
		log.Warn("Cannot search for next match: current line is not loaded.")
		return
	}

	var start int
	if reverse {
		start = m.offset
	} else {
		start = m.fwd[0].nextOffset()
	}

	m.longFileOpInProgress = true
	m.cancelLongFileOp.Reset()
	m.msg = ""

	log.Info("Searching for next regexp match: regexp=%q", re)

	go FindMatch(r, m, start, re, reverse)
}

func FindMatch(r Reactor, m *Model, start int, re *regexp.Regexp, reverse bool) {
	defer r.Enque(func() { m.longFileOpInProgress = false }, "find match complete")

	var lineReader LineReader
	if reverse {
		lineReader = NewBackwardLineReader(m.content, start)
	} else {
		lineReader = NewForwardLineReader(m.content, start)
	}

	offset := start
	for {
		if m.cancelLongFileOp.Cancelled() {
			return
		}
		line, err := lineReader.ReadLine()
		if err != nil {
			if err != io.EOF {
				r.Stop(fmt.Errorf("Could not read: error=%v", err))
				return
			} else {
				r.Enque(func() {
					msg := "regex search complete: no match found"
					setMessage(m, msg)
				}, "no match found")
				return
			}
		}
		if reverse {
			offset -= len(line)
		}
		if re.Match(line) {
			break
		}
		if !reverse {
			offset += len(line)
		}
	}

	r.Enque(func() {
		log.Info("Regexp search completed with match.")
		moveToOffset(m, offset)
	}, "match found")
}

func currentRE(m *Model) *regexp.Regexp {
	re := m.tmpRegex
	if re == nil && len(m.regexes) > 0 {
		re = m.regexes[0].re
	}
	return re
}
