package main

import (
	"bytes"
	"io"
	"time"
)

type Screen interface {
	Write(cells []byte, cols int)
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

	writer io.Writer

	reactor Reactor
	log     Logger
}

func (t *termScreen) Write(cells []byte, cols int) {

	t.log.Info("Preparing screen write contents.")

	// Calculate byte sequence to send to terminal.
	// TODO: Diff algorithm.
	t.nextWrite.Reset()
	t.nextWrite.WriteString("\x1b[2J\x1b[H")
	for _, b := range cells {
		assert(b >= 32 && b <= 126) // Char must be visible.
		t.nextWrite.WriteByte(b)
	}

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
