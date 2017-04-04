package main

import (
	"bytes"
	"fmt"
	"io"
	"time"
)

type Style byte

const (
	fgMask      Style = 0x07
	fgIsSetMask Style = 0x08
	bgMask      Style = 0x70
	bgIsSetMask Style = 0x80
)

func (s Style) withFG(fg Style) Style {
	s |= fgIsSetMask
	s &= ^fgMask
	s |= fg
	return s
}

func (s Style) withBG(fg Style) Style {
	s |= bgIsSetMask
	s &= ^bgMask
	s |= fg << 4
	return s
}

func (s Style) hasFG() bool {
	return (s & fgIsSetMask) != 0
}

func (s Style) hasBG() bool {
	return (s & bgIsSetMask) != 0
}

func (s Style) fg() int {
	return int(30 + (s & fgMask))
}

func (s Style) bg() int {
	return int(40 + ((s & bgMask) >> 4))
}

func (s Style) escapeCode() string {
	if !s.hasFG() && !s.hasBG() {
		return "\x1b[m" // SGR0
	}
	var out string
	if s.hasFG() {
		out += fmt.Sprintf("\x1b[%dm", s.fg())
	}
	if s.hasBG() {
		out += fmt.Sprintf("\x1b[%dm", s.bg())
	}
	return out
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
