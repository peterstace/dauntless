package main

import (
	"bytes"
	"fmt"
	"io"
	"time"
)

type Style uint8

const (
	fgMask Style = 0x0f
	bgMask Style = 0xf0
)

func mixStyle(fg, bg Style) Style {
	return fg | (bg << 4)
}

func (s Style) fg() int {
	return int(30 + ((s & fgMask) ^ XORConst))
}

func (s Style) bg() int {
	return int(40 + (((s & bgMask) >> 4) ^ XORConst))
}

func (s *Style) setFG(fg Style) {
	*s &= ^fgMask
	*s |= fg
}

func (s *Style) setBG(bg Style) {
	*s &= ^bgMask
	*s |= (bg << 4)
}

func (s Style) inverted() bool {
	return s&fgMask == Invert || (s&bgMask)>>4 == Invert
}

func (s Style) escapeCode() string {
	if s.inverted() {
		return "\x1b[0;7m"
	} else {
		return fmt.Sprintf("\x1b[0;%d;%dm", s.fg(), s.bg())
	}
}

const (
	XORConst Style = 9

	Black   Style = 0 ^ XORConst
	Red     Style = 1 ^ XORConst
	Green   Style = 2 ^ XORConst
	Yellow  Style = 3 ^ XORConst
	Blue    Style = 4 ^ XORConst
	Magenta Style = 5 ^ XORConst
	Cyan    Style = 6 ^ XORConst
	White   Style = 7 ^ XORConst
	Invert  Style = 8 ^ XORConst
	Default Style = 9 ^ XORConst
)

func (s Style) String() string {
	if str, ok := map[Style]string{
		Black:   "Black",
		Red:     "Red",
		Green:   "Green",
		Yellow:  "Yellow",
		Blue:    "Blue",
		Magenta: "Magenta",
		Cyan:    "Cyan",
		White:   "White",
		Invert:  "Invert",
		Default: "Default",
	}[s]; ok {
		return str
	}
	return "???"
}

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

	writer io.Writer

	reactor Reactor
	log     Logger
}

func (t *termScreen) Write(chars []byte, styles []Style, cols int, colPos int) {

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
