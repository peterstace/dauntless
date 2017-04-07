package main

import (
	"os"
	"regexp"
)

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

type regex struct {
	style Style
	re    *regexp.Regexp
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

	regexes []regex

	overlay bool // XXX: For debugging.
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
	a.fillScreenBuffer()
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

		'0': func() { a.setFG(Black) },
		'1': func() { a.setFG(Red) },
		'2': func() { a.setFG(Green) },
		'3': func() { a.setFG(Yellow) },
		'4': func() { a.setFG(Blue) },
		'5': func() { a.setFG(Magenta) },
		'6': func() { a.setFG(Cyan) },
		'7': func() { a.setFG(White) },
		'8': func() { a.setFG(Invert) },
		'9': func() { a.setFG(Default) },

		')': func() { a.setBG(Black) },
		'!': func() { a.setBG(Red) },
		'@': func() { a.setBG(Green) },
		'#': func() { a.setBG(Yellow) },
		'$': func() { a.setBG(Blue) },
		'%': func() { a.setBG(Magenta) },
		'^': func() { a.setBG(Cyan) },
		'&': func() { a.setBG(White) },
		'*': func() { a.setBG(Invert) },
		'(': func() { a.setBG(Default) },

		// XXX For debugging:
		's': func() {
			a.overlay = !a.overlay
			a.refresh()
		},
	}[b]

	if !ok {
		a.log.Info("Unhandled key press: %d", b)
		return
	}

	fn()
}

func (a *app) quit() {
	a.log.Info("Quitting.")
	a.reactor.Stop(nil)
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
			a.reactor.Stop(err)
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
	a.fillScreenBuffer()
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
	a.fillScreenBuffer()
}

func (a *app) setFG(s Style) {
	a.log.Info("Setting FG: %v", s)
	if len(a.regexes) > 0 {
		a.regexes[0].style.setFG(s)
	}
	a.refresh()
}

func (a *app) setBG(s Style) {
	a.log.Info("Setting BG: %v", s)
	if len(a.regexes) > 0 {
		a.regexes[0].style.setBG(s)
	}
	a.refresh()
}

func (a *app) startSearchCommand() {
	a.commandMode = search
	a.log.Info("Accepting search command.")
	a.refresh()
}

func (a *app) finishSearchCommand() {
	a.log.Info("Search command entered: %q", a.commandText)
	re, err := regexp.Compile(a.commandText)
	if err != nil {
		a.log.Warn("Could not compile regexp: Regexp=%q Err=%q", a.commandText, err)
		return
	}
	a.log.Info("Regex compiled.")
	if len(a.regexes) == 0 {
		a.regexes = []regex{regex{}}
	}
	a.regexes[0] = regex{mixStyle(White, Red), re}
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

const (
	backLoadFactor    = 1
	forwardLoadFactor = 2
)

func (a *app) fillScreenBuffer() {
	if a.needsLoadingForward() {
		a.loadForward()
	} else if a.needsLoadingBackward() {
		a.loadBackward()
	} else {
		a.log.Info("Screen buffer didn't need filling.")
	}

	// TODO: Screen buffer trimming.
}

func (a *app) needsLoadingForward() bool {
	if len(a.fwd) >= a.rows*forwardLoadFactor {
		return false
	}
	if len(a.fwd) > 0 {
		lastLine := a.fwd[len(a.fwd)-1]
		if lastLine.offset+len(lastLine.data) == a.fileSize {
			return false
		}
	}
	return true
}

func (a *app) needsLoadingBackward() bool {
	if len(a.bck) >= a.rows*backLoadFactor {
		return false
	}
	if len(a.bck) > 0 {
		lastLine := a.bck[len(a.bck)-1]
		if lastLine.offset == 0 {
			return false
		}
	}
	return true
}

func (a *app) loadForward() {
	a.log.Debug("Loading forward.")
	offset := a.offset
	if len(a.fwd) > 0 {
		lastLine := a.fwd[len(a.fwd)-1]
		offset = lastLine.offset + len(lastLine.data)
	}
	a.loader.Load(LoadRequest{
		Offset:   offset,
		Amount:   defaultLoadAmount,
		Forwards: true,
	})
}

func (a *app) loadBackward() {
	a.log.Debug("Loading backward.")
	end := a.offset
	if len(a.bck) > 0 {
		end = a.bck[len(a.bck)-1].offset
	}
	start := end - defaultLoadAmount
	start = max(0, start)
	a.loader.Load(LoadRequest{
		Offset:   start,
		Amount:   end - start,
		Forwards: false,
	})
}

func (a *app) TermSize(rows, cols int, err error) {
	if a.rows != rows || a.cols != cols {
		a.rows = rows
		a.cols = cols
		a.log.Info("Term size: rows=%d cols=%d", rows, cols)

		// Since the terminal changed size, we may now need to have a different
		// number of lines loaded into the screen buffer.
		a.fillScreenBuffer()
		a.refresh()
	}
}

func (a *app) refresh() {

	if a.rows < 0 || a.cols < 0 {
		a.log.Warn("Can't refresh, don't know term size yet")
		return
	}

	a.log.Info("Refreshing")

	dim := a.rows * a.cols
	if len(a.screenBuffer) != dim {
		a.screenBuffer = make([]byte, dim)
	}
	if len(a.stylesBuffer) != dim {
		a.stylesBuffer = make([]Style, dim)
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
		a.stylesBuffer[i] = mixStyle(Default, Default)
	}

	assert(len(a.fwd) == 0 || a.fwd[0].offset == a.offset)
	offset := a.offset
	lineRows := a.rows - 2 // 2 rows reserved for status line and command line.
	for row := 0; row < lineRows; row++ {
		if row < len(a.fwd) {
			bounds := make([][][2]int, len(a.regexes))
			matches := make([][][]int, len(a.regexes))
			for r := range a.regexes {
				matches[r] = a.regexes[r].re.FindAllStringIndex(a.fwd[row].data, -1)
				bounds[r] = make([][2]int, len(matches[r]))
			}
			col := 0
			for i := 0; col+1 < a.cols && i < len(a.fwd[row].data); i++ {
				for r := range matches {
					for j := range matches[r] {
						if matches[r][j][0] == i {
							bounds[r][j][0] = col
						}
						if matches[r][j][1] == i {
							bounds[r][j][1] = col
						}
					}
				}
				col += writeByte(a.screenBuffer[row*a.cols+col:(row+1)*a.cols], a.fwd[row].data[i], col)
			}
			for r := range bounds {
				for j := range bounds[r] {
					for col := bounds[r][j][0]; col < bounds[r][j][1]; col++ {
						a.stylesBuffer[row*a.cols+col] = a.regexes[r].style
					}
				}
			}
		} else if len(a.fwd) != 0 && a.fwd[len(a.fwd)-1].offset+len(a.fwd[len(a.fwd)-1].data) >= a.fileSize {
			// Reached end of file.
			assert(a.fwd[len(a.fwd)-1].offset+len(a.fwd[len(a.fwd)-1].data) == a.fileSize) // Assert that it's actually equal.
			break
		} else {
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

	// XXX: Debugging.
	if a.overlay {
		overlaySwatch(a.screenBuffer, a.stylesBuffer, a.cols)
	}

	a.screen.Write(a.screenBuffer, a.stylesBuffer, a.cols)
}

const defaultLoadAmount = 64

func (a *app) LoadComplete(resp LoadResponse) {

	req := resp.Request

	a.log.Info("Data loaded: From=%d To=%d Len=%d",
		req.Offset, req.Offset+len(resp.Payload), len(resp.Payload))
	if resp.FileSize != a.fileSize {
		a.log.Info("File size changed: oldSize=%d newSize=%d", a.fileSize, resp.FileSize)
		a.fileSize = resp.FileSize
	}
	a.fileSize = resp.FileSize

	lines := extractLines(resp.Payload)
	offset := req.Offset
	for _, data := range lines {
		if len(a.fwd) == 0 && offset == a.offset {
			a.fwd = append(a.fwd, line{offset, data})
		} else if len(a.fwd) > 0 && a.fwd[len(a.fwd)-1].offset+len(a.fwd[len(a.fwd)-1].data) == offset {
			a.fwd = append(a.fwd, line{offset, data})
		}
		offset += len(data)
	}
	for i := len(lines) - 1; i >= 0; i-- {
		data := lines[i]
		if len(a.bck) == 0 && offset == a.offset {
			a.bck = append(a.bck, line{offset - len(data), data})
		} else if len(a.bck) > 0 && a.bck[len(a.bck)-1].offset == offset {
			a.bck = append(a.bck, line{offset - len(data), data})
		}
		offset -= len(data)
	}

	if len(lines) > 0 {
		a.refresh()
	} else {
		if req.Forwards {
			reachedEndOfFile := req.Offset+len(resp.Payload) >= a.fileSize
			if reachedEndOfFile {
				a.log.Warn("Data loaded didn't contain at least one complete line: reached EOF")
			} else {
				a.log.Warn("Data loaded didn't contain at least one complete line: retrying with double amount.")
				req.Amount *= 2
				a.loader.Load(req)
			}
		} else {
			reachedStartOfFile := req.Offset == 0
			if reachedStartOfFile {
				a.log.Warn("Data loaded didn't contain at least one complete line: reached offset 0")
			} else {
				a.log.Warn("Data loaded didn't contain at least one complete line: retrying with double amount.")
				end := req.Offset + req.Amount
				req.Amount *= 2
				req.Offset = end - req.Amount
				if req.Offset < 0 {
					req.Amount += req.Offset // reduces req.Amount
					req.Offset = 0
				}
				a.loader.Load(req)
			}
		}
	}

	a.fillScreenBuffer()
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

func overlaySwatch(chars []byte, styles []Style, cols int) {

	// 10 by 10 swatches. 4 chars wide each. So 40 wide and 10 high. Plus a
	// border of 1 around each side.

	const width = 42
	const height = 12

	rows := len(chars) / cols
	startCol := (cols - width) / 2
	startRow := (rows - height) / 2

	// Draw the border.
	inv := mixStyle(Invert, Invert)
	for col := startCol; col < startCol+width; col++ {
		chars[startRow*cols+col] = ' '
		styles[startRow*cols+col] = inv
		chars[(startRow+height-1)*cols+col] = ' '
		styles[(startRow+height-1)*cols+col] = inv
	}
	for row := startRow; row < startRow+height; row++ {
		chars[row*cols+startCol] = ' '
		styles[row*cols+startCol] = inv
		chars[row*cols+startCol+width-1] = ' '
		styles[row*cols+startCol+width-1] = inv
	}

	for fg := 0; fg <= 9; fg++ {
		for bg := 0; bg <= 9; bg++ {
			r := startRow + 1 + int(fg)
			c := startCol + 2 + 4*int(bg)
			chars[r*cols+c] = '0' + byte(fg)
			chars[r*cols+c+1] = '0' + byte(bg)
			chars[r*cols+c-1] = ' '
			chars[r*cols+c+2] = ' '
			style := mixStyle(Style(fg), Style(bg))
			for i := 0; i < 4; i++ {
				styles[r*cols+c-1+i] = style
			}
		}
	}

}
