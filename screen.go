package main

import (
	"bytes"
	"fmt"
	"io"
	"time"
)

type Style byte // 0x0f masks FG, 0xf0 masks BG.

func (s Style) escapeCode() string {
	return fmt.Sprintf(
		"\x1b[%dm\x1b[%dm",
		30+s&0x0f,
		40+(s&0xf0)>>4,
	)
}

func fgAndBg(fg Style, bg Style) Style {
	return fg | (bg >> 4)
}

const (
	Black   Style = 0
	Red     Style = 1
	Green   Style = 2
	Yellow  Style = 3
	Blue    Style = 4
	Magenta Style = 5
	Cyan    Style = 6
	White   Style = 7
)

type Screen interface {
	Write(chars []byte, styles []Style, cols int)
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

func (t *termScreen) Write(chars []byte, styles []Style, cols int) {

	t.log.Info("Preparing screen write contents.")

	assert(len(chars) == len(styles))

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
