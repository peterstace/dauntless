package main

import "os"

type App interface {
	Initialise()
	KeyPress(byte)
	TermSize(rows, cols int, err error)
	LoadComplete(LoadResponse)
	Signal(os.Signal)
}

type command int

const (
	none command = iota
	search
)

type line struct {
	offset int
	data   string
}

type app struct {
	reactor Reactor
	log     Logger

	filename string

	loader Loader

	rows, cols int

	// Invariants:
	//  1) If fwd is populated, then offset will match the first line.
	//  2) Fwd and bck contain consecutive lines.
	offset int
	fwd    []line
	bck    []line

	fileSize int

	stylesBuffer []Style
	screenBuffer []byte
	screen       Screen

	commandMode command
	commandText string
}

func NewApp(reactor Reactor, filename string, loader Loader, logger Logger, screen Screen) App {
	return &app{
		reactor:  reactor,
		filename: filename,
		loader:   loader,
		log:      logger,
		rows:     -1,
		cols:     -1,
		screen:   screen,
	}
}

func (a *app) Initialise() {
	a.log.Info("***************** Initialising log viewer ******************")
}

func (a *app) Signal(sig os.Signal) {
	a.log.Info("Caught signal: %v", sig)
	a.quit()
}

func (a *app) KeyPress(b byte) {

	a.log.Info("Key press: %q", string([]byte{b}))

	if a.commandMode != none {
		a.consumeCommandChar(b)
		return
	}

	fn, ok := map[byte]func(){
		'q': a.quit,
		'j': a.moveDownBySingleLine,
		'k': a.moveUpBySingleLine,
		'd': a.moveDownByHalfScreen,
		'u': a.moveUpByHalfScreen,
		'r': a.repaint,
		'R': a.discardBufferedInputAndRepaint,
		'g': a.moveTop,
		'G': a.moveBottom,
		'/': a.startSearchCommand,
	}[b]

	if !ok {
		a.log.Info("Unhandled key press: %d", b)
		return
	}

	fn()
}

func (a *app) quit() {
	a.log.Info("Quitting.")
	a.reactor.Stop()
}

func (a *app) moveDownBySingleLine() {
	a.moveDown()
	a.refresh()
}

func (a *app) moveUpBySingleLine() {
	a.moveUp()
	a.refresh()
}

func (a *app) moveDownByHalfScreen() {
	for i := 0; i < a.rows/2; i++ {
		a.moveDown()
	}
	a.refresh()
}

func (a *app) moveUpByHalfScreen() {
	for i := 0; i < a.rows/2; i++ {
		a.moveUp()
	}
	a.refresh()
}

func (a *app) repaint() {
	a.log.Info("Repainting screen.")
	a.refresh()
}

func (a *app) discardBufferedInputAndRepaint() {
	a.log.Info("Discarding buffered input and repainting screen.")
	a.fwd = nil
	a.bck = nil
	a.refresh()
}

func (a *app) moveDown() {

	a.log.Info("Moving down.")

	if len(a.fwd) == 0 {
		a.log.Warn("Cannot move down: current line not loaded.")
		return
	}

	ln := a.fwd[0]
	newOffset := ln.offset + len(ln.data)

	if newOffset == a.fileSize {
		a.log.Info("Cannot move down: reached EOF.")
		return
	}

	a.moveToOffset(newOffset)
}

func (a *app) moveUp() {

	a.log.Info("Moving up.")

	if a.offset == 0 {
		a.log.Info("Cannot move back: at start of file.")
		return
	}

	if len(a.bck) == 0 {
		a.log.Warn("Cannot move back: previous line not loaded.")
		return
	}

	a.moveToOffset(a.bck[0].offset)
}

func (a *app) moveTop() {
	a.log.Info("Jumping to start of file.")
	a.moveToOffset(0)
	a.refresh()
}

func (a *app) moveBottom() {

	a.log.Info("Jumping to bottom of file.")

	go func() {
		offset, err := FindJumpToBottomOffset(a.filename)
		if err != nil {
			a.log.Warn("Could not find jump-to-bottom offset: %v", err)
			a.reactor.Stop()
			return
		}
		a.reactor.Enque(func() {
			a.moveToOffset(offset)
			a.refresh()
		})
	}()
}

func (a *app) moveToOffset(offset int) {
	a.log.Info("Moving to offset: currentOffset=%d newOffset=%d", a.offset, offset)

	assert(offset >= 0)
	assert(offset < a.fileSize)

	if a.offset == offset {
		a.log.Info("Already at target offset.")
	} else if offset < a.offset {
		a.moveUpToOffset(offset)
	} else {
		a.moveDownToOffset(offset)
	}
}

func (a *app) moveUpToOffset(offset int) {

	a.log.Info("Moving up to offset: currentOffset=%d newOffset=%d", a.offset, offset)

	haveTargetLoaded := false
	for _, ln := range a.bck {
		if ln.offset == offset {
			haveTargetLoaded = true
			break
		}
	}
	if haveTargetLoaded {
		for a.offset != offset {
			ln := a.bck[0]
			a.fwd = append([]line{ln}, a.fwd...)
			a.bck = a.bck[1:]
			a.offset = ln.offset
		}
	} else {
		a.fwd = nil
		a.bck = nil
		a.offset = offset
	}
}

func (a *app) moveDownToOffset(offset int) {

	a.log.Info("Moving down to offset: currentOffset=%d newOffset=%d", a.offset, offset)

	haveTargetLoaded := false
	for _, ln := range a.fwd {
		if ln.offset == offset {
			haveTargetLoaded = true
			break
		}
	}
	if haveTargetLoaded {
		for a.offset != offset {
			ln := a.fwd[0]
			a.fwd = a.fwd[1:]
			a.bck = append([]line{ln}, a.bck...)
			a.offset = ln.offset + len(ln.data)
		}
	} else {
		a.fwd = nil
		a.bck = nil
		a.offset = offset
	}
}

func (a *app) startSearchCommand() {
	a.commandMode = search
	a.log.Info("Accepting search command.")
	a.refresh()
}

func (a *app) finishSearchCommand() {
	a.log.Info("Search command entered: %q", a.commandText)
	// TODO:
}

func (a *app) consumeCommandChar(b byte) {

	assert(a.commandMode != none)

	if b >= ' ' && b <= '~' {
		a.commandText += string([]byte{b})
		a.log.Info("Added to command: text=%q", a.commandText)
	} else if b == 127 {
		if len(a.commandText) >= 1 {
			a.commandText = a.commandText[:len(a.commandText)-1]
			a.log.Info("Backspacing char from command text: text=%q", a.commandText)
		} else {
			a.log.Info("Cannot backspace from empty command text.")
		}
	} else if b == 10 {
		a.log.Info("Finished command mode.")
		switch a.commandMode {
		case search:
			a.finishSearchCommand()
		case none:
			assert(false)
		default:
			assert(false)
		}
		a.commandMode = none
		a.commandText = ""
	} else {
		a.log.Warn("Refusing to add char to command: %d", b)
	}

	a.refresh()
}

func (a *app) TermSize(rows, cols int, err error) {
	if a.rows != rows || a.cols != cols {
		a.rows = rows
		a.cols = cols
		a.log.Info("Term size: rows=%d cols=%d", rows, cols)
		a.refresh()
	}
}

func (a *app) refresh() {

	if a.rows < 0 || a.cols < 0 {
		a.log.Warn("Can't refresh, don't know term size yet")
		return
	}

	a.log.Info("Refreshing")

	if len(a.screenBuffer) != a.rows*a.cols {
		a.screenBuffer = make([]byte, a.rows*a.cols)
	}
	a.renderScreen()
}

func writeByte(buf []byte, b byte, offsetInLine int) int {

	// Normal chars.
	if b >= 32 && b <= 126 { // ' ' up to '~'
		if len(buf) >= 1 {
			buf[0] = b
			return 1
		}
		return 0
	}

	// Special cases.
	switch b {
	case '\n':
		return 0
	case '\t':
		const tabSize = 4
		spaces := tabSize - offsetInLine%tabSize
		for i := 0; i < spaces && i < len(buf); i++ {
			buf[i] = ' '
		}
		return min(spaces, len(buf))
	}

	// Unknown chars.
	if len(buf) >= 1 {
		buf[0] = '.'
		return 1
	}
	return 0
}

func (a *app) renderScreen() {

	a.log.Info("Rendering screen.")

	for i := range a.screenBuffer {
		a.screenBuffer[i] = ' '
	}

	assert(len(a.fwd) == 0 || a.fwd[0].offset == a.offset)
	offset := a.offset
	lineRows := a.rows - 2 // 2 rows reserved for status line and command line.
	for row := 0; row < lineRows; row++ {
		if row < len(a.fwd) {
			col := 0
			for i := 0; col+1 < a.cols && i < len(a.fwd[row].data); i++ {
				col += writeByte(a.screenBuffer[row*a.cols+col:(row+1)*a.cols], a.fwd[row].data[i], col)
			}
		} else if len(a.fwd) != 0 && a.fwd[len(a.fwd)-1].offset+len(a.fwd[len(a.fwd)-1].data) >= a.fileSize {
			// Reached end of file.
			assert(a.fwd[len(a.fwd)-1].offset+len(a.fwd[len(a.fwd)-1].data) == a.fileSize) // Assert that it's actually equal.
			break
		} else {
			a.loader.Load(offset, defaultLoadAmount)
			buildLoadingScreen(a.screenBuffer, a.cols)
			break
		}
		offset += len(a.fwd[row].data)
	}

	commandLineText := ""
	switch a.commandMode {
	case search:
		commandLineText = "Enter search regexp: " + a.commandText
	case none:
	default:
		assert(false)
	}
	commandRow := a.rows - 1
	copy(a.screenBuffer[commandRow*a.cols:(commandRow+1)*a.cols], commandLineText)

	if len(a.stylesBuffer) != len(a.screenBuffer) {
		a.stylesBuffer = make([]Style, len(a.screenBuffer))
	}
	for i := range a.stylesBuffer {
		a.stylesBuffer[i] = Style(0)
	}
	a.screen.Write(a.screenBuffer, a.stylesBuffer, a.cols)
}

const defaultLoadAmount = 64

func (a *app) LoadComplete(resp LoadResponse) {

	a.log.Info("Data loaded: From=%d To=%d Len=%d", resp.Offset, resp.Offset+len(resp.Payload), len(resp.Payload))
	if resp.FileSize != a.fileSize {
		a.log.Info("File size changed: oldSize=%d newSize=%d", a.fileSize, resp.FileSize)
		a.fileSize = resp.FileSize
	}
	a.fileSize = resp.FileSize

	offset := resp.Offset
	containedLine := false
	for _, data := range extractLines(resp.Payload) {
		containedLine = true
		if len(a.fwd) == 0 && offset == a.offset {
			a.fwd = append(a.fwd, line{offset, data})
		} else if len(a.fwd) > 0 && a.fwd[len(a.fwd)-1].offset+len(a.fwd[len(a.fwd)-1].data) == offset {
			a.fwd = append(a.fwd, line{offset, data})
		}
		offset += len(data)
	}

	if containedLine {
		a.refresh()
	} else if reachedEndOfFile := resp.Offset+len(resp.Payload) == a.fileSize; !reachedEndOfFile {
		a.log.Warn("Data loaded didn't contain at least one complete line: retrying with double amount.")
		a.loader.Load(resp.Offset, 2*len(resp.Payload))
	} else {
		a.log.Warn("Data loaded didn't contain at least one complete line: reached EOF")
	}
}

func buildLoadingScreen(buf []byte, cols int) {
	for i := range buf {
		buf[i] = ' '
	}
	const loading = "Loading..."
	row := len(buf) / cols / 2
	startCol := (cols - len(loading)) / 2
	copy(buf[row*cols+startCol:], loading)
}
