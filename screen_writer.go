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

func NewTermScreen(w io.Writer) Screen {
	t := &termScreen{
		writer:          w,
		inboundWriteCh:  make(chan ScreenState),
		writeCompleteCh: make(chan struct{}),
	}
	go t.run()
	return t
}

type termScreen struct {
	writer io.Writer

	inboundWriteCh  chan ScreenState
	writeCompleteCh chan struct{}

	writeInProgress  bool
	hasPending       bool
	pendingState     ScreenState
	lastWrittenState ScreenState
}

func (t *termScreen) Write(state ScreenState, force bool) {
	t.inboundWriteCh <- state
}

func (t *termScreen) run() {
	var ()
	for {
		select {
		case state := <-t.inboundWriteCh:
			t.newState(state)
		case <-t.writeCompleteCh:
			t.writeComplete()
		}
	}
}

func (t *termScreen) newState(state ScreenState) {
	t.hasPending = true
	state.CloneInto(&t.pendingState)
	// TODO if force, then set last written state to empty
	if t.writeInProgress {
		return
	}
	t.outputPending()
}

func (t *termScreen) writeComplete() {
	t.writeInProgress = false
	if t.hasPending {
		t.outputPending()
	}
}

func (t *termScreen) outputPending() {
	assert(t.hasPending)
	assert(!t.writeInProgress)
	t.hasPending = false

	diff := ScreenDiff(t.lastWrittenState, t.pendingState)
	if diff.Len() == 0 {
		return
	}

	t.writeInProgress = true
	t.pendingState.CloneInto(&t.lastWrittenState)

	go func() {
		io.Copy(t.writer, diff)
		// Stops flashing under constant scroll by putting an artificial delay
		// between updating the screen.
		time.Sleep(10 * time.Millisecond)
		t.writeCompleteCh <- struct{}{}
	}()
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
