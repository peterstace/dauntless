package main

import (
	"bytes"
	"fmt"
	"io"
	"time"
)

type Screen interface {
	Write(state ScreenState, force bool)
}

func NewTermScreen(w io.Writer, r Reactor, log Logger) Screen {
	return &termScreen{
		writer:  w,
		reactor: r,
		log:     log,
	}
}

type termScreen struct {
	writeInProgress  bool
	hasPending       bool
	pendingState     ScreenState
	lastWrittenState ScreenState
	writer           io.Writer
	reactor          Reactor
	log              Logger
}

func (t *termScreen) Write(state ScreenState, force bool) {

	t.log.Info("Preparing screen write contents.")

	t.hasPending = true
	state.CloneInto(&t.pendingState)
	if force {
		t.lastWrittenState.Cols = 0
		t.lastWrittenState.Chars = nil
		t.lastWrittenState.Styles = nil
	}

	if t.writeInProgress {
		t.log.Info("Write already in progress, will write after completion.")
		return
	}

	t.outputPending()
}

func ScreenDiff(from, to ScreenState) *bytes.Buffer {

	renderAll := len(from.Chars) != len(to.Chars) || from.Cols != to.Cols

	buf := new(bytes.Buffer)

	writtenStyle := false
	var currentStyle Style

	for row := 0; row < to.Rows(); row++ {
		firstMismatchCol := -1
		if renderAll {
			firstMismatchCol = to.Cols - 1
		} else {
			for col := to.Cols - 1; col >= 0; col-- {
				idx := to.RowColIdx(row, col)
				if to.Chars[idx] != from.Chars[idx] || to.Styles[idx] != from.Styles[idx] {
					firstMismatchCol = col
					break
				}
			}
		}
		if firstMismatchCol >= 0 {
			fmt.Fprintf(buf, "\x1b[%d;H", row+1)
			for col := 0; col <= firstMismatchCol; col++ {
				idx := to.RowColIdx(row, col)
				if !writtenStyle || currentStyle != to.Styles[idx] {
					buf.WriteString(to.Styles[idx].escapeCode())
					writtenStyle = true
					currentStyle = to.Styles[idx]
				}
				buf.WriteByte(to.Chars[idx])
			}
		}
	}

	if buf.Len() > 0 || from.ColPos != to.ColPos {
		fmt.Fprintf(buf, "\x1b[%d;%dH", to.Rows(), to.ColPos+1)
	}
	return buf
}

func (t *termScreen) outputPending() {

	assert(t.hasPending)
	assert(!t.writeInProgress)
	t.hasPending = false

	diff := ScreenDiff(t.lastWrittenState, t.pendingState)
	if diff.Len() == 0 {
		t.log.Info("Screen state is the same, aborting write.")
		return
	}

	t.log.Info("Writing to screen: bytes=%d", diff.Len())
	t.writeInProgress = true
	t.pendingState.CloneInto(&t.lastWrittenState)

	go func() {
		io.Copy(t.writer, diff)

		// TODO: Tweak to stop "flashing" under constant scroll. Should
		// probably be variable/parameter.
		time.Sleep(100 * time.Millisecond)

		t.reactor.Enque(t.writeComplete)
	}()
}

func (t *termScreen) writeComplete() {

	t.log.Info("Screen write complete: hasPending=%t", t.hasPending)

	t.writeInProgress = false
	if t.hasPending {
		t.outputPending()
	}
}
