package main

import (
	"bytes"
	"fmt"
	"io"
	"time"
)

type Screen interface {
	Write(ScreenState)
}

func NewTermScreen(w io.Writer, r Reactor, log Logger) Screen {
	return &termScreen{
		currentWrite: new(bytes.Buffer),
		nextWrite:    new(bytes.Buffer),
		writer:       w,
		reactor:      r,
		log:          log,
	}
}

type termScreen struct {
	writeInProgress bool
	pendingWrite    bool
	currentWrite    *bytes.Buffer
	nextWrite       *bytes.Buffer

	last ScreenState

	writer io.Writer

	reactor Reactor
	log     Logger
}

func (t *termScreen) Write(state ScreenState) {

	t.log.Info("Preparing screen write contents.")

	if t.last.Equal(state) {
		t.log.Info("No change to screen, aborting write.")
		return
	}
	state.CloneInto(&t.last)

	// Calculate byte sequence to send to terminal.
	// TODO: Diff algorithm.
	t.nextWrite.Reset()
	t.nextWrite.WriteString("\x1b[H")
	currentStyle := state.Styles[0]
	for i := range state.Chars {
		if i == 0 || state.Styles[i] != currentStyle {
			currentStyle = state.Styles[i]
			t.nextWrite.WriteString(currentStyle.escapeCode())
		}
		assert(state.Chars[i] >= 32 && state.Chars[i] <= 126) // Char must be visible.
		t.nextWrite.WriteByte(state.Chars[i])
	}
	fmt.Fprintf(t.nextWrite, "\x1b[%d;%dH", len(state.Chars)/state.Cols+1, state.ColPos+1)

	if t.writeInProgress {
		t.log.Info("Write already in progress, will write after completion.")
		t.pendingWrite = true
		return
	}

	t.outputToScreen()
}

func (t *termScreen) outputToScreen() {

	assert(!t.writeInProgress)
	assert(!t.pendingWrite)

	t.writeInProgress = true
	t.nextWrite, t.currentWrite = t.currentWrite, t.nextWrite

	t.log.Info("Writing to screen: bytes=%d", t.currentWrite.Len())

	go func() {
		io.Copy(t.writer, t.currentWrite)

		// TODO: Tweak to stop "flashing" under constant scroll. Should
		// probably be variable/parameter.
		time.Sleep(100 * time.Millisecond)

		t.reactor.Enque(t.writeComplete)
	}()
}

func (t *termScreen) writeComplete() {

	t.log.Info("Screen write complete: pendingWrite=%t", t.pendingWrite)

	t.writeInProgress = false
	if t.pendingWrite {
		t.pendingWrite = false
		t.outputToScreen()
	}
}
