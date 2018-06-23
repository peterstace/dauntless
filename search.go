package main

import (
	"fmt"
	"io"
	"regexp"
)

func (a *app) jumpToMatch(reverse bool) {
	re := currentRE(&a.model)
	if re == nil {
		msg := "no regex to jump to"
		log.Info(msg)
		a.setMessage(msg)
		return
	}

	if len(a.model.fwd) == 0 {
		log.Warn("Cannot search for next match: current line is not loaded.")
		return
	}

	var start int
	if reverse {
		start = a.model.offset
	} else {
		start = a.model.fwd[0].nextOffset()
	}

	a.model.longFileOpInProgress = true
	a.model.cancelLongFileOp.Reset()
	a.model.msg = ""

	log.Info("Searching for next regexp match: regexp=%q", re)

	go a.FindMatch(start, re, reverse)
}

func (a *app) FindMatch(start int, re *regexp.Regexp, reverse bool) {
	defer a.reactor.Enque(func() { a.model.longFileOpInProgress = false }, "find match complete")

	var lineReader LineReader
	if reverse {
		lineReader = NewBackwardLineReader(a.model.content, start)
	} else {
		lineReader = NewForwardLineReader(a.model.content, start)
	}

	offset := start
	for {
		if a.model.cancelLongFileOp.Cancelled() {
			return
		}
		line, err := lineReader.ReadLine()
		if err != nil {
			if err != io.EOF {
				a.reactor.Stop(fmt.Errorf("Could not read: error=%v", err))
				return
			} else {
				a.reactor.Enque(func() {
					msg := "regex search complete: no match found"
					a.setMessage(msg)
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

	a.reactor.Enque(func() {
		log.Info("Regexp search completed with match.")
		moveToOffset(&a.model, offset)
	}, "match found")
}

func currentRE(m *Model) *regexp.Regexp {
	re := m.tmpRegex
	if re == nil && len(m.regexes) > 0 {
		re = m.regexes[0].re
	}
	return re
}
