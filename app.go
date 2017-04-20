package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"
)

type App interface {
	Initialise()
	KeyPress(Key)
	TermSize(rows, cols int, err error)
	FileSize(int, error)
	Signal(os.Signal)
}

type command int

const (
	none command = iota
	search
	colour
	seek
	bisect
)

type line struct {
	offset int
	data   string
}

func (l line) nextOffset() int {
	return l.offset + len(l.data)
}

type regex struct {
	style Style
	re    *regexp.Regexp
}

type app struct {
	reactor Reactor
	log     Logger
	config  Config

	filename string

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

	tmpRegex *regexp.Regexp
	regexes  []regex

	lineWrapMode bool
	xPosition    int

	fillingScreenBuffer bool

	msg      string
	msgSetAt time.Time
}

func NewApp(reactor Reactor, filename string, logger Logger, screen Screen, config Config) App {
	return &app{
		reactor:  reactor,
		filename: filename,
		log:      logger,
		config:   config,
		screen:   screen,
	}
}

func (a *app) Initialise() {
	a.log.Info("***************** Initialising log viewer ******************")
	a.fillScreenBuffer()
}

func (a *app) Signal(sig os.Signal) {
	a.log.Info("Caught signal: %v", sig)
	if a.commandMode == none {
		a.quit()
	} else {
		a.log.Info("Cancelling command.")
		a.msg = "" // Don't want old message to show up.
		a.commandMode = none
		a.commandText = ""
		a.refresh()
	}
}

func (a *app) KeyPress(k Key) {

	a.log.Info("Key press: %s", k)

	if a.commandMode != none {
		a.consumeCommandKey(k)
		return
	}

	fn, ok := map[Key]func(){
		"q": a.quit,

		"j": a.moveDownBySingleLine,
		"k": a.moveUpBySingleLine,
		"d": a.moveDownByHalfScreen,
		"u": a.moveUpByHalfScreen,

		DownArrowKey: a.moveDownBySingleLine,
		UpArrowKey:   a.moveUpBySingleLine,
		PageDownKey:  a.moveDownByHalfScreen,
		PageUpKey:    a.moveUpByHalfScreen,

		LeftArrowKey:  a.reduceXPosition,
		RightArrowKey: a.increaseXPosition,

		"r": a.repaint,
		"R": a.discardBufferedInputAndRepaint,

		"g": a.moveTop,
		"G": a.moveBottom,

		"/": a.startSearchCommand,
		"n": a.jumpToNextMatch,
		"N": a.jumpToPrevMatch,

		"w": a.toggleLineWrapMode,

		"c":  a.startColourCommand,
		"\t": a.cycleRegexp,
		"x":  a.deleteRegexp,

		"s": a.startSeekCommand,
		"b": a.startBisectCommand,
	}[k]

	if !ok {
		a.log.Info("Key press was unhandled.")
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
	a.fillScreenBuffer()
}

func (a *app) moveUpBySingleLine() {
	a.moveUp()
	a.refresh()
	a.fillScreenBuffer()
}

func (a *app) moveDownByHalfScreen() {
	for i := 0; i < a.rows/2; i++ {
		a.moveDown()
	}
	a.refresh()
	a.fillScreenBuffer()
}

func (a *app) moveUpByHalfScreen() {
	for i := 0; i < a.rows/2; i++ {
		a.moveUp()
	}
	a.refresh()
	a.fillScreenBuffer()
}

func (a *app) repaint() {
	a.log.Info("Repainting screen.")
	a.refresh()
}

func (a *app) discardBufferedInputAndRepaint() {
	a.log.Info("Discarding buffered input and repainting screen.")
	a.fwd = nil
	a.bck = nil
	a.fillScreenBuffer()
}

func (a *app) moveDown() {
	a.log.Info("Moving down.")
	if len(a.fwd) < 2 {
		a.log.Warn("Cannot move down: reason=\"not enough lines loaded\" linesLoaded=%d", len(a.fwd))
		return
	}
	a.moveToOffset(a.fwd[1].offset)
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
	a.fillScreenBuffer()
}

func (a *app) moveBottom() {

	a.log.Info("Jumping to bottom of file.")

	go func() {
		offset, err := FindJumpToBottomOffset(a.filename)
		a.reactor.Enque(func() {
			if err != nil {
				a.log.Warn("Could not find jump-to-bottom offset: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.moveToOffset(offset)
			a.refresh()
			a.fillScreenBuffer()
		})
	}()
}

func (a *app) moveToOffset(offset int) {
	a.log.Info("Moving to offset: currentOffset=%d newOffset=%d", a.offset, offset)

	assert(offset >= 0)

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
	re, err := regexp.Compile(a.commandText)
	if err != nil {
		a.log.Warn("Could not compile regexp: Regexp=%q Err=%q", a.commandText, err)
		a.setMessage(err.Error())
		return
	}
	a.log.Info("Regex compiled.")
	a.tmpRegex = re
}

func (a *app) consumeCommandKey(k Key) {

	assert(a.commandMode != none)

	if len(k) != 1 {
		a.log.Warn("Ignoring multi char sequence.")
		return
	}

	b := k[0]
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
	} else if b == '\n' {
		a.log.Info("Finished command mode.")
		a.msg = "" // Don't want old message to show up after the command.
		switch a.commandMode {
		case search:
			a.finishSearchCommand()
		case colour:
			a.finishColourCommand()
		case seek:
			a.finishSeekCommand()
		case bisect:
			a.finishBisectCommand()
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

func (a *app) jumpToNextMatch() {

	re := a.currentRE()
	if re == nil {
		msg := "no regex to jump to"
		a.log.Info(msg)
		a.setMessage(msg)
		return
	}

	if len(a.fwd) == 0 {
		a.log.Warn("Cannot search for next match: current line is not loaded.")
		return
	}
	startOffset := a.fwd[0].nextOffset()

	a.log.Info("Searching for next regexp match: regexp=%q", re)

	go func() {
		offset, err := FindNextMatch(a.filename, startOffset, re)
		a.reactor.Enque(func() {
			if err == io.EOF {
				msg := "regex search complete: no match found"
				a.log.Info(msg)
				a.setMessage(msg)
				return
			}
			if err != nil {
				a.log.Warn("Regexp search completed with error: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.log.Info("Regexp search completed with match.")
			a.moveToOffset(offset)
			a.refresh()
			a.fillScreenBuffer()
		})
	}()
}

func (a *app) jumpToPrevMatch() {

	re := a.currentRE()
	if re == nil {
		msg := "no regex to jump to"
		a.log.Info(msg)
		a.setMessage(msg)
		return
	}

	endOffset := a.offset

	a.log.Info("Searching for previous regexp match: regexp=%q", re)

	go func() {
		offset, err := FindPrevMatch(a.filename, endOffset, re)
		a.reactor.Enque(func() {
			if err != nil && err != io.EOF {
				a.log.Warn("Regexp search completed with error: %v", err)
				a.reactor.Stop(err)
				return
			}
			if err == io.EOF {
				msg := "regex search complete: no match found"
				a.log.Info(msg)
				a.setMessage(msg)
				return
			}
			a.log.Info("Regexp search completed with match.")
			a.moveToOffset(offset)
			a.refresh()
			a.fillScreenBuffer()
		})
	}()
}

func (a *app) toggleLineWrapMode() {
	if a.lineWrapMode {
		a.log.Info("Toggling out of line wrap mode.")
	} else {
		a.log.Info("Toggling into line wrap mode.")
	}
	a.lineWrapMode = !a.lineWrapMode
	a.xPosition = 0
	a.refresh()
}

func (a *app) startColourCommand() {
	if a.currentRE() == nil {
		msg := "cannot select regex color: no active regex"
		a.log.Warn(msg)
		a.setMessage(msg)
		return
	}
	a.commandMode = colour
	a.log.Info("Accepting colour command.")
	a.refresh()
}

func (a *app) cycleRegexp() {

	if len(a.regexes) == 0 {
		msg := "no regexes to cycle between"
		a.log.Warn(msg)
		a.setMessage(msg)
		return
	}

	a.tmpRegex = nil // Any temp re gets discarded.
	a.regexes = append(a.regexes[1:], a.regexes[0])
	a.refresh()
}

func (a *app) deleteRegexp() {
	if a.tmpRegex != nil {
		a.tmpRegex = nil
	} else if len(a.regexes) > 0 {
		a.regexes = a.regexes[1:]
	} else {
		msg := "no regexes to delete"
		a.log.Warn(msg)
		a.setMessage(msg)
	}
	a.refresh()
}

var styles = [...]Style{Default, Black, Red, Green, Yellow, Blue, Magenta, Cyan, White}

func (a *app) finishColourCommand() {

	a.log.Info("Colour command entered: %q", a.commandText)

	style, err := parseColourCode(a.commandText)
	if err != nil {
		a.log.Warn("Could not parse entered colour: %v", err)
		a.setMessage(err.Error())
		return
	}
	a.log.Info("Style parsed.")

	if a.tmpRegex != nil {
		a.regexes = append([]regex{{style, a.tmpRegex}}, a.regexes...)
		a.tmpRegex = nil
	} else if len(a.regexes) > 0 {
		a.regexes[0].style = style
	} else {
		// Should not have been allowed to start the colour command.
		assert(false)
	}

	a.refresh()
}

func parseColourCode(code string) (Style, error) {
	err := fmt.Errorf("colour code must be in format [0-8][0-8]: %v", code)
	if len(code) != 2 {
		return 0, err
	}
	fg := code[0]
	bg := code[1]
	if fg < '0' || fg > '8' || bg < '0' || bg > '8' {
		return 0, err
	}
	return mixStyle(styles[fg-'0'], styles[bg-'0']), nil
}

func (a *app) startSeekCommand() {
	a.commandMode = seek
	a.log.Info("Accepting seek command.")
	a.refresh()
}

func (a *app) finishSeekCommand() {

	a.log.Info("Seek command entered: %q", a.commandText)

	seekPct, err := strconv.ParseFloat(a.commandText, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse seek percentage: %v", err)
		a.log.Warn(msg)
		a.setMessage(msg)
		return
	}

	if seekPct < 0 || seekPct > 100 {
		msg := fmt.Sprintf("Seek percentage out of range [0, 100]: %v", seekPct)
		a.log.Warn(msg)
		a.setMessage(msg)
		return
	}

	go func() {
		offset, err := FindSeekOffset(a.filename, seekPct)
		a.reactor.Enque(func() {
			if err != nil {
				a.log.Warn("Could to find start of line at offset: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.moveToOffset(offset)
			a.refresh()
			a.fillScreenBuffer()
		})
	}()
}

func (a *app) startBisectCommand() {
	a.commandMode = bisect
	a.log.Info("Accepting bisect command.")
	a.refresh()
}

func (a *app) finishBisectCommand() {
	a.log.Info("Bisect command entered: %q", a.commandText)
	target := a.commandText
	go func() {
		offset, err := Bisect(a.filename, target, a.config.BisectMask)
		a.reactor.Enque(func() {
			if err != nil {
				a.log.Warn("Could not find bisect target: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.moveToOffset(offset)
			a.refresh()
			a.fillScreenBuffer()
		})
	}()
}

func (a *app) reduceXPosition() {
	a.changeXPosition(max(0, a.xPosition-a.cols/4))
}

func (a *app) increaseXPosition() {
	a.changeXPosition(max(0, a.xPosition+a.cols/4))
}

func (a *app) changeXPosition(newPosition int) {
	a.log.Info("Changing x position: old=%v new=%v", a.xPosition, newPosition)
	if a.xPosition != newPosition {
		a.xPosition = newPosition
		a.refresh()
	}
}

const (
	backLoadFactor      = 1
	forwardLoadFactor   = 2
	backUnloadFactor    = 2
	forwardUnloadFactor = 3
)

func (a *app) fillScreenBuffer() {

	if a.fillingScreenBuffer {
		a.log.Info("Aborting filling screen buffer, already in progress.")
		return
	}

	a.log.Info("Filling screen buffer, has initial state: fwd=%d bck=%d", len(a.fwd), len(a.bck))

	if lines := a.needsLoadingForward(); lines != 0 {
		a.loadForward(lines)
	} else if lines := a.needsLoadingBackward(); lines != 0 {
		a.loadBackward(lines)
	} else {
		a.log.Info("Screen buffer didn't need filling.")
	}

	// Prune buffers.
	neededFwd := min(len(a.fwd), a.rows*forwardUnloadFactor)
	a.fwd = a.fwd[:neededFwd]
	neededBck := min(len(a.bck), a.rows*backUnloadFactor)
	a.bck = a.bck[:neededBck]
}

func (a *app) needsLoadingForward() int {
	if len(a.fwd) >= a.rows*forwardLoadFactor {
		return 0
	}
	if len(a.fwd) > 0 {
		lastLine := a.fwd[len(a.fwd)-1]
		if lastLine.offset+len(lastLine.data) >= a.fileSize {
			return 0
		}
	}
	return a.rows*forwardLoadFactor - len(a.fwd)
}

func (a *app) needsLoadingBackward() int {
	if a.offset == 0 {
		return 0
	}
	if len(a.bck) >= a.rows*backLoadFactor {
		return 0
	}
	if len(a.bck) > 0 {
		lastLine := a.bck[len(a.bck)-1]
		if lastLine.offset == 0 {
			return 0
		}
	}
	return a.rows*backLoadFactor - len(a.bck)
}

func (a *app) loadForward(amount int) {

	offset := a.offset
	if len(a.fwd) > 0 {
		offset = a.fwd[len(a.fwd)-1].nextOffset()
	}
	a.log.Debug("Loading forward: offset=%d amount=%d", offset, amount)

	a.fillingScreenBuffer = true
	go func() {
		lines, err := LoadFwd(a.filename, offset, amount)
		a.reactor.Enque(func() {
			if err != nil {
				a.log.Warn("Error loading forward: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.log.Debug("Got fwd lines: numLines=%d initialFwd=%d initialBck=%d", len(lines), len(a.fwd), len(a.bck))
			for _, data := range lines {
				if (len(a.fwd) == 0 && offset == a.offset) ||
					(len(a.fwd) > 0 && a.fwd[len(a.fwd)-1].nextOffset() == offset) {
					a.fwd = append(a.fwd, line{offset, data})
				}
				offset += len(data)
			}
			a.log.Debug("After adding to data structure: fwd=%d bck=%d", len(a.fwd), len(a.bck))
			// TODO: Does it make sense to have this conditional?
			if len(lines) > 0 {
				a.refresh()
			}
			a.fillingScreenBuffer = false
			a.fillScreenBuffer()
		})
	}()
}

func (a *app) loadBackward(amount int) {

	offset := a.offset
	if len(a.bck) > 0 {
		offset = a.bck[len(a.bck)-1].offset
	}
	a.log.Debug("Loading backward: offset=%d amount=%d", offset, amount)

	a.fillingScreenBuffer = true
	go func() {
		lines, err := LoadBck(a.filename, offset, amount)
		a.reactor.Enque(func() {
			if err != nil {
				a.log.Warn("Error loading backward: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.log.Debug("Got bck lines: numLines=%d initialFwd=%d initialBck=%d", len(lines), len(a.fwd), len(a.bck))
			for _, data := range lines {
				if (len(a.bck) == 0 && offset == a.offset) ||
					(len(a.bck) > 0 && a.bck[len(a.bck)-1].offset == offset) {
					a.bck = append(a.bck, line{offset - len(data), data})
				}
				offset -= len(data)
			}
			a.log.Debug("After adding to data structure: fwd=%d bck=%d", len(a.fwd), len(a.bck))
			// TODO: Does it make sense to have this conditional?
			if len(lines) > 0 {
				a.refresh()
			}
			a.fillingScreenBuffer = false
			a.fillScreenBuffer()
		})
	}()
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

func (a *app) FileSize(size int, err error) {
	oldSize := a.fileSize
	if size != oldSize {
		a.fileSize = size
		a.log.Info("File size changed: old=%d new=%d", oldSize, size)
		a.fillScreenBuffer()
	}
}

func (a *app) refresh() {

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

func displayByte(b byte) byte {
	assert(b != '\n')
	switch {
	case b >= 32 && b < 126:
		return b
	case b == '\t':
		return ' '
	default:
		return '?'
	}
}

func (a *app) clearScreenBuffers() {
	for i := range a.screenBuffer {
		a.screenBuffer[i] = ' '
		a.stylesBuffer[i] = mixStyle(Default, Default)
	}
}

func (a *app) renderScreen() {

	a.log.Info("Rendering screen.")

	a.clearScreenBuffers()

	assert(len(a.fwd) == 0 || a.fwd[0].offset == a.offset)
	var lineBuf []byte
	var styleBuf []Style
	var fwdIdx int
	lineRows := a.rows - 2 // 2 rows reserved for status line and command line.
	for row := 0; row < lineRows; row++ {
		if fwdIdx < len(a.fwd) {
			usePrefix := len(lineBuf) != 0
			if len(lineBuf) == 0 {
				assert(len(styleBuf) == 0)
				data := a.fwd[fwdIdx].data
				assert(data[len(data)-1] == '\n')
				data = data[:len(data)-1]
				lineBuf = a.renderLine(data)
				styleBuf = a.renderStyle(data)
				fwdIdx++
			}
			if !a.lineWrapMode {
				if a.xPosition < len(lineBuf) {
					copy(a.screenBuffer[row*a.cols:(row+1)*a.cols], lineBuf[a.xPosition:])
					copy(a.stylesBuffer[row*a.cols:(row+1)*a.cols], styleBuf[a.xPosition:])
				}
				lineBuf = nil
				styleBuf = nil
			} else {
				var prefix string
				if usePrefix && len(a.config.WrapPrefix)+1 < a.cols {
					prefix = a.config.WrapPrefix
				}
				copy(a.screenBuffer[row*a.cols:(row+1)*a.cols], prefix)
				copiedA := copy(a.screenBuffer[row*a.cols+len(prefix):(row+1)*a.cols], lineBuf)
				copiedB := copy(a.stylesBuffer[row*a.cols+len(prefix):(row+1)*a.cols], styleBuf)
				assert(copiedA == copiedB)
				lineBuf = lineBuf[copiedA:]
				styleBuf = styleBuf[copiedB:]
			}
		} else if len(a.fwd) != 0 && a.fwd[len(a.fwd)-1].nextOffset() >= a.fileSize {
			// Reached end of file. `a.fileSize` may be slightly out of date,
			// however next time it's updated the additional lines will be
			// displayed.
			a.screenBuffer[a.rowColIdx(row, 0)] = '~'
		} else {
			a.clearScreenBuffers()
			buildLoadingScreen(a.screenBuffer, a.cols)
			break
		}
	}

	a.drawStatusLine()

	commandLineText := ""
	switch a.commandMode {
	case search:
		commandLineText = "Enter search regexp (interrupt to cancel): " + a.commandText
	case colour:
		commandLineText = "Enter colour code (interrupt to cancel): " + a.commandText
	case seek:
		commandLineText = "Enter seek percentage (interrupt to cancel): " + a.commandText
	case bisect:
		commandLineText = "Enter bisect target (interrupt to cancel): " + a.commandText
	case none:
		if time.Now().Sub(a.msgSetAt) < msgLingerDuration {
			commandLineText = a.msg
		}
	default:
		assert(false)
	}
	commandRow := a.rows - 1
	copy(a.screenBuffer[commandRow*a.cols:(commandRow+1)*a.cols], commandLineText)

	if a.commandMode == colour {
		a.overlaySwatch()
	}

	col := a.cols - 1
	if a.commandMode != none {
		col = min(col, len(commandLineText))
	}
	a.screen.Write(a.screenBuffer, a.stylesBuffer, a.cols, col)
}

func (a *app) renderLine(data string) []byte {
	buf := make([]byte, len(data))
	for i := range data {
		buf[i] = displayByte(data[i])
	}
	return buf
}

func (a *app) renderStyle(data string) []Style {

	regexes := a.regexes
	if a.tmpRegex != nil {
		regexes = append(regexes, regex{mixStyle(Invert, Invert), a.tmpRegex})
	}
	buf := make([]Style, len(data))
	for _, regex := range regexes {
		for _, match := range regex.re.FindAllStringIndex(data, -1) {
			for i := match[0]; i < match[1]; i++ {
				buf[i] = regex.style
			}
		}
	}
	return buf
}

func (a *app) drawStatusLine() {

	statusRow := a.rows - 2
	for col := 0; col < a.cols; col++ {
		a.stylesBuffer[statusRow*a.cols+col] = mixStyle(Invert, Invert)
	}

	// Offset percentage.
	pct := float64(a.offset) / float64(a.fileSize) * 100
	var pctStr string
	switch {
	case pct < 10:
		// 9.99%
		pctStr = fmt.Sprintf("%3.2f%%", pct)
	default:
		// 99.9%
		pctStr = fmt.Sprintf("%3.1f%%", pct)
	}

	// Line wrap mode.
	var lineWrapMode string
	if a.lineWrapMode {
		lineWrapMode = "line-wrap-mode:on "
	} else {
		lineWrapMode = "line-wrap-mode:off"
	}

	currentRegexpStr := "re:<none>"
	if a.tmpRegex != nil {
		currentRegexpStr = "re(tmp):" + a.tmpRegex.String()
	} else if len(a.regexes) > 0 {
		currentRegexpStr = fmt.Sprintf("re(%d):%s", len(a.regexes), a.regexes[0].re.String())
	}

	statusRight := fmt.Sprintf("fwd:%d bck:%d ", len(a.fwd), len(a.bck)) + lineWrapMode + " " + pctStr + " "
	statusLeft := " " + a.filename + " " + currentRegexpStr

	buf := a.screenBuffer[statusRow*a.cols : (statusRow+1)*a.cols]
	copy(buf[len(buf)-len(statusRight):], statusRight)
	copy(buf[:], statusLeft)
}

func buildLoadingScreen(buf []byte, cols int) {
	const loading = "Loading..."
	row := len(buf) / cols / 2
	startCol := (cols - len(loading)) / 2
	copy(buf[row*cols+startCol:], loading)
}

func (a *app) overlaySwatch() {

	const sideBorder = 2
	const topBorder = 1
	const colourWidth = 4
	const swatchWidth = len(styles)*colourWidth + sideBorder*2
	const swatchHeight = len(styles) + topBorder*2

	startCol := (a.cols - swatchWidth) / 2
	startRow := (a.rows - swatchHeight) / 2
	endCol := startCol + swatchWidth
	endRow := startRow + swatchHeight

	for row := startRow; row < endRow; row++ {
		for col := startCol; col < endCol; col++ {
			idx := a.rowColIdx(row, col)
			if col-startCol < 2 || endCol-col <= 2 || row-startRow < 1 || endRow-row <= 1 {
				a.stylesBuffer[idx] = mixStyle(Invert, Invert)
			}
			a.screenBuffer[idx] = ' '
		}
	}

	for fg := 0; fg < len(styles); fg++ {
		for bg := 0; bg < len(styles); bg++ {
			start := startCol + sideBorder + bg*colourWidth
			row := startRow + topBorder + fg
			a.screenBuffer[a.rowColIdx(row, start+1)] = byte(fg) + '0'
			a.screenBuffer[a.rowColIdx(row, start+2)] = byte(bg) + '0'
			style := mixStyle(styles[fg], styles[bg])
			for i := 0; i < 4; i++ {
				a.stylesBuffer[a.rowColIdx(row, start+i)] = style
			}
		}
	}
}

func (a *app) rowColIdx(row, col int) int {
	return row*a.cols + col
}

func (a *app) currentRE() *regexp.Regexp {
	re := a.tmpRegex
	if re == nil && len(a.regexes) > 0 {
		re = a.regexes[0].re
	}
	return re
}

const msgLingerDuration = 5 * time.Second

func (a *app) setMessage(msg string) {
	a.log.Info("Setting message: %q", msg)
	a.msg = msg
	a.msgSetAt = time.Now()
	a.refresh()
	go func() {
		time.Sleep(msgLingerDuration)
		a.refresh()
	}()
}
