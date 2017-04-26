package main

import (
	"bytes"
	"fmt"
	"io"
	"time"
)

type Screen interface {
	Write(chars []byte, styles []Style, cols int, colPos int)
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

	lastCols   int
	lastChars  []byte
	lastStyles []Style
	lastColPos int

	writer io.Writer

	reactor Reactor
	log     Logger
}

func (t *termScreen) Write(chars []byte, styles []Style, cols int, colPos int) {

	t.log.Info("Preparing screen write contents.")

	assert(len(chars) == len(styles))

	same := true
	if colPos != t.lastCols {
		same = false
	}
	if t.lastCols != cols || len(chars) != len(t.lastChars) {
		same = false
		t.lastChars = make([]byte, len(chars))
		t.lastStyles = make([]Style, len(styles))
	} else {
		for i := range chars {
			if t.lastChars[i] != chars[i] || t.lastStyles[i] != styles[i] {
				same = false
				break
			}
		}
	}
	if same {
		t.log.Info("No change to screen, aborting write.")
		return
	}
	t.lastCols = cols
	t.lastColPos = colPos
	copy(t.lastChars, chars)
	copy(t.lastStyles, styles)

	// Calculate byte sequence to send to terminal.
	// TODO: Diff algorithm.
	t.nextWrite.Reset()
	t.nextWrite.WriteString("\x1b[H")
	currentStyle := styles[0]
	for i := range chars {
		if i == 0 || styles[i] != currentStyle {
			currentStyle = styles[i]
			t.nextWrite.WriteString(currentStyle.escapeCode())
		}
		assert(chars[i] >= 32 && chars[i] <= 126) // Char must be visible.
		t.nextWrite.WriteByte(chars[i])
	}
	fmt.Fprintf(t.nextWrite, "\x1b[%d;%dH", len(chars)/cols+1, colPos+1)

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
