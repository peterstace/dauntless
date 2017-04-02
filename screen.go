package main

import (
	"bytes"
	"io"
)

type Screen interface {
	Write(cells []byte, cols int)
}

func NewTermScreen(w io.Writer, r Reactor) Screen {
	return &termScreen{
		currentWrite: new(bytes.Buffer),
		nextWrite:    new(bytes.Buffer),
		writer:       w,
		reactor:      r,
	}
}

type termScreen struct {
	writeInProgress bool
	pendingWrite    bool
	currentWrite    *bytes.Buffer
	nextWrite       *bytes.Buffer

	writer io.Writer

	reactor Reactor
}

func (t *termScreen) Write(cells []byte, cols int) {

	// Calculate byte sequence to send to terminal.
	// TODO: Diff algorithm.
	t.nextWrite.Reset()
	t.nextWrite.WriteString("\x1b[2J\x1b[H")
	for _, b := range cells {
		assert(b >= 32 && b <= 126) // Char must be visible.
		t.nextWrite.WriteByte(b)
	}

	if t.writeInProgress {
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
	go func() {
		io.Copy(t.writer, t.currentWrite)
		t.reactor.Enque(t.writeComplete)
	}()
}

func (t *termScreen) writeComplete() {

	t.writeInProgress = false
	if t.pendingWrite {
		t.pendingWrite = false
		t.outputToScreen()
	}
}
