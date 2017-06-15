package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"
)

func jumpToNextMatch(m *Model, r Reactor) {

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
	startOffset := m.fwd[0].nextOffset()

	m.longFileOpInProgress = true
	m.cancelLongFileOp.Reset()

	log.Info("Searching for next regexp match: regexp=%q", re)

	go FindNextMatch(&m.cancelLongFileOp, r, m, startOffset, re)
}

func jumpToPrevMatch(m *Model, r Reactor) {

	re := currentRE(m)
	if re == nil {
		msg := "no regex to jump to"
		log.Info(msg)
		setMessage(m, msg)
		return
	}

	endOffset := m.offset

	m.longFileOpInProgress = true
	m.cancelLongFileOp.Reset()

	log.Info("Searching for previous regexp match: regexp=%q", re)

	go FindPrevMatch(&m.cancelLongFileOp, r, m, endOffset, re)
}

func FindNextMatch(cancel *Cancellable, r Reactor, m *Model, start int, re *regexp.Regexp) {

	defer r.Enque(func() { m.longFileOpInProgress = false })

	f, err := os.Open(m.filename)
	if err != nil {
		r.Stop(fmt.Errorf("Could not open file: %v", err))
		return
	}
	defer f.Close()

	time.Sleep(500 * time.Millisecond)

	if _, err := f.Seek(int64(start), 0); err != nil {
		r.Stop(fmt.Errorf("Could not seek: offset=%d", start))
		return
	}

	reader := bufio.NewReader(f)
	offset := start
	for {
		if cancel.Cancelled() {
			return
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				r.Stop(fmt.Errorf("Could not read: error=%v", err))
				return
			} else {
				r.Enque(func() {
					msg := "regex search complete: no match found"
					setMessage(m, msg)
				})
				return
			}
		}
		if re.Match(line) {
			break
		}
		offset += len(line)
	}

	r.Enque(func() {
		log.Info("Regexp search completed with match.")
		moveToOffset(m, offset)
	})
}

func FindPrevMatch(cancel *Cancellable, r Reactor, m *Model, endOffset int, re *regexp.Regexp) {

	defer r.Enque(func() { m.longFileOpInProgress = false })

	f, err := os.Open(m.filename)
	if err != nil {
		r.Stop(fmt.Errorf("Could not open file: %v", err))
		return
	}
	defer f.Close()

	time.Sleep(500 * time.Millisecond)

	lineReader := NewBackwardLineReader(f, endOffset)
	offset := endOffset
	for {
		if cancel.Cancelled() {
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
				})
				return
			}
		}
		offset -= len(line)
		if re.Match(line) {
			break
		}
	}

	r.Enque(func() {
		log.Info("Regexp search completed with match.")
		moveToOffset(m, offset)
	})
}

func currentRE(m *Model) *regexp.Regexp {
	re := m.tmpRegex
	if re == nil && len(m.regexes) > 0 {
		re = m.regexes[0].re
	}
	return re
}
