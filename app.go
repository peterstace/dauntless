package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"time"
)

type App interface {
	Initialise()
	KeyPress(Key)
	TermSize(rows, cols int, err error)
	FileSize(int)
	Signal(os.Signal)

	CommandFailed(error)
	SearchCommandEntered(*regexp.Regexp)
	ColourCommandEntered(Style)
	SeekCommandEntered(pct float64)
	BisectCommandEntered(target string)
	QuitCommandEntered(bool)
}

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

	screen Screen

	dataMissing     bool
	dataMissingFrom time.Time

	command     CommandMode
	commandText string
	commandPos  int

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
	a.reactor.SetPostHook(func() {
		a.fillScreenBuffer()
		a.refresh()
	})
}

func (a *app) Signal(sig os.Signal) {
	a.log.Info("Caught signal: %v", sig)
	if a.command == nil {
		a.startQuitCommand()
	} else {
		a.log.Info("Cancelling command.")
		a.msg = "" // Don't want old message to show up.
		a.command = nil
		a.commandText = ""
		a.commandPos = 0
	}
}

func (a *app) KeyPress(k Key) {

	a.log.Info("Key press: %s", k)

	if a.command != nil {
		a.consumeCommandKey(k)
		return
	}

	fn, ok := map[Key]func(){
		"q": a.startQuitCommand,

		"j": a.moveDown,
		"k": a.moveUp,
		"d": a.moveDownByHalfScreen,
		"u": a.moveUpByHalfScreen,

		DownArrowKey: a.moveDown,
		UpArrowKey:   a.moveUp,
		PageDownKey:  a.moveDownByHalfScreen,
		PageUpKey:    a.moveUpByHalfScreen,

		LeftArrowKey:  a.reduceXPosition,
		RightArrowKey: a.increaseXPosition,

		"r": a.discardBufferedInputAndRepaint,

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

func (a *app) moveDownByHalfScreen() {
	for i := 0; i < a.rows/2; i++ {
		a.moveDown()
	}
}

func (a *app) moveUpByHalfScreen() {
	for i := 0; i < a.rows/2; i++ {
		a.moveUp()
	}
}

func (a *app) discardBufferedInputAndRepaint() {
	a.log.Info("Discarding buffered input and repainting screen.")
	a.fwd = nil
	a.bck = nil
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

func (a *app) CommandFailed(err error) {
	a.log.Warn("Command failed: %v", err)
	a.setMessage(err.Error())
}

func (a *app) startSearchCommand() {
	a.command = search{}
	a.log.Info("Accepting search command.")
}

func (a *app) SearchCommandEntered(re *regexp.Regexp) {
	a.tmpRegex = re
}

func (a *app) startColourCommand() {
	if a.currentRE() == nil {
		msg := "cannot select regex color: no active regex"
		a.log.Warn(msg)
		a.setMessage(msg)
		return
	}
	a.command = colour{}
	a.log.Info("Accepting colour command.")
}

func (a *app) ColourCommandEntered(style Style) {
	if a.tmpRegex != nil {
		a.regexes = append([]regex{{style, a.tmpRegex}}, a.regexes...)
		a.tmpRegex = nil
	} else if len(a.regexes) > 0 {
		a.regexes[0].style = style
	} else {
		// Should not have been allowed to start the colour command.
		assert(false)
	}
}

func (a *app) startSeekCommand() {
	a.command = seek{}
	a.log.Info("Accepting seek command.")
}

func (a *app) SeekCommandEntered(pct float64) {
	go func() {
		offset, err := FindSeekOffset(a.filename, pct)
		a.reactor.Enque(func() {
			if err != nil {
				a.log.Warn("Could to find start of line at offset: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.moveToOffset(offset)
		})
	}()
}

func (a *app) startBisectCommand() {
	a.command = bisect{}
	a.log.Info("Accepting bisect command.")
}

func (a *app) BisectCommandEntered(target string) {
	a.log.Info("Bisect command entered: %q", a.commandText)
	go func() {
		offset, err := Bisect(a.filename, target, a.config.BisectMask)
		a.reactor.Enque(func() {
			if err != nil {
				a.log.Warn("Could not find bisect target: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.moveToOffset(offset)
		})
	}()
}

func (a *app) startQuitCommand() {
	a.command = quit{}
	a.log.Info("Accepting quit command.")
}

func (a *app) QuitCommandEntered(quit bool) {
	if quit {
		a.reactor.Stop(nil)
	}
}

func (a *app) consumeCommandKey(k Key) {

	assert(a.command != nil)

	if len(k) == 1 {
		b := k[0]
		if b >= ' ' && b <= '~' {
			a.commandText = a.commandText[:a.commandPos] + string([]byte{b}) + a.commandText[a.commandPos:]
			a.commandPos++
		} else if b == 127 && len(a.commandText) >= 1 {
			a.commandText = a.commandText[:a.commandPos-1] + a.commandText[a.commandPos:]
			a.commandPos--
		} else if b == '\n' {
			a.log.Info("Finished command mode.")
			a.msg = "" // Don't want old message to show up after the command.
			assert(a.command != nil)
			a.command.Entered(a.commandText, a)
			a.command = nil
			a.commandText = ""
			a.commandPos = 0
		}
	} else {
		if k == LeftArrowKey {
			a.commandPos = max(0, a.commandPos-1)
		} else if k == RightArrowKey {
			a.commandPos = min(a.commandPos+1, len(a.commandText))
		} else if k == DeleteKey && a.commandPos < len(a.commandText) {
			a.commandText = a.commandText[:a.commandPos] + a.commandText[a.commandPos+1:]
		}
	}
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
	if a.fileSize == 0 {
		return 0
	}
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
			a.fillingScreenBuffer = false
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
			a.fillingScreenBuffer = false
		})
	}()
}

func (a *app) TermSize(rows, cols int, err error) {
	if a.rows != rows || a.cols != cols {
		a.rows = rows
		a.cols = cols
		a.log.Info("Term size: rows=%d cols=%d", rows, cols)
	}
}

func (a *app) FileSize(size int) {
	oldSize := a.fileSize
	if size != oldSize {
		a.fileSize = size
		a.log.Info("File size changed: old=%d new=%d", oldSize, size)
	}
}

func (a *app) refresh() {
	a.log.Info("Refreshing")
	if a.cols == 0 || a.rows == 0 {
		a.log.Info("Aborting refresh: rows=%d cols=%d", a.rows, a.cols)
		return
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

const loadingScreenGrace = 200 * time.Millisecond

func (a *app) renderScreen() {

	a.log.Info("Rendering screen.")

	state := NewScreenState(a.rows, a.cols)
	state.Init()

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
					copy(state.Chars[row*a.cols:(row+1)*a.cols], lineBuf[a.xPosition:])
					copy(state.Styles[row*a.cols:(row+1)*a.cols], styleBuf[a.xPosition:])
				}
				lineBuf = nil
				styleBuf = nil
			} else {
				var prefix string
				if usePrefix && len(a.config.WrapPrefix)+1 < a.cols {
					prefix = a.config.WrapPrefix
				}
				copy(state.Chars[row*a.cols:(row+1)*a.cols], prefix)
				copiedA := copy(state.Chars[row*a.cols+len(prefix):(row+1)*a.cols], lineBuf)
				copiedB := copy(state.Styles[row*a.cols+len(prefix):(row+1)*a.cols], styleBuf)
				assert(copiedA == copiedB)
				lineBuf = lineBuf[copiedA:]
				styleBuf = styleBuf[copiedB:]
			}
			a.dataMissing = false
		} else if a.fileSize == 0 || len(a.fwd) != 0 && a.fwd[len(a.fwd)-1].nextOffset() >= a.fileSize {
			// Reached end of file. `a.fileSize` may be slightly out of date,
			// however next time it's updated the additional lines will be
			// displayed.
			state.Chars[state.RowColIdx(row, 0)] = '~'
			a.dataMissing = false
		} else if a.dataMissing && time.Now().Sub(a.dataMissingFrom) > loadingScreenGrace {
			// Haven't been able to display any data for at least the grace
			// period, so display the loading screen instead.
			buildLoadingScreen(state)
			break
		} else {
			// Cannot display the data, but within the grace period. Abort the
			// display procedure, trying again after the grace period.
			a.dataMissing = true
			a.dataMissingFrom = time.Now()
			go func() {
				time.Sleep(loadingScreenGrace)
				a.reactor.Enque(func() {})
			}()
			return
		}
	}

	a.drawStatusLine(state)

	col := a.cols - 1
	commandLineText := ""
	if a.command != nil {
		prompt := a.command.Prompt()
		commandLineText = prompt + a.commandText
		col = min(col, len(prompt)+a.commandPos)
	} else {
		if time.Now().Sub(a.msgSetAt) < msgLingerDuration {
			commandLineText = a.msg
		}
	}

	commandRow := a.rows - 1
	copy(state.Chars[commandRow*a.cols:(commandRow+1)*a.cols], commandLineText)

	if a.command == (colour{}) {
		overlaySwatch(state)
	}

	a.screen.Write(state)
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
		regexes = append(regexes, regex{MixStyle(Invert, Invert), a.tmpRegex})
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

func (a *app) drawStatusLine(state ScreenState) {

	statusRow := a.rows - 2
	for col := 0; col < state.Cols; col++ {
		state.Styles[statusRow*a.cols+col] = MixStyle(Invert, Invert)
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

	buf := state.Chars[statusRow*a.cols : (statusRow+1)*a.cols]
	copy(buf[max(0, len(buf)-len(statusRight)):], statusRight)
	copy(buf[:], statusLeft)
}

func buildLoadingScreen(state ScreenState) {
	state.Init() // Clear anything previously set.
	const loading = "Loading..."
	row := state.Rows() / 2
	startCol := (state.Cols - len(loading)) / 2
	copy(state.Chars[row*state.Cols+startCol:], loading)
}

func overlaySwatch(state ScreenState) {

	const sideBorder = 2
	const topBorder = 1
	const colourWidth = 4
	const swatchWidth = len(styles)*colourWidth + sideBorder*2
	const swatchHeight = len(styles) + topBorder*2

	startCol := (state.Cols - swatchWidth) / 2
	startRow := (state.Rows() - swatchHeight) / 2
	endCol := startCol + swatchWidth
	endRow := startRow + swatchHeight

	for row := startRow; row < endRow; row++ {
		for col := startCol; col < endCol; col++ {
			idx := state.RowColIdx(row, col)
			if col-startCol < 2 || endCol-col <= 2 || row-startRow < 1 || endRow-row <= 1 {
				state.Styles[idx] = MixStyle(Invert, Invert)
			}
			state.Chars[idx] = ' '
		}
	}

	for fg := 0; fg < len(styles); fg++ {
		for bg := 0; bg < len(styles); bg++ {
			start := startCol + sideBorder + bg*colourWidth
			row := startRow + topBorder + fg
			state.Chars[state.RowColIdx(row, start+1)] = byte(fg) + '0'
			state.Chars[state.RowColIdx(row, start+2)] = byte(bg) + '0'
			style := MixStyle(styles[fg], styles[bg])
			for i := 0; i < 4; i++ {
				state.Styles[state.RowColIdx(row, start+i)] = style
			}
		}
	}
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
	go func() {
		time.Sleep(msgLingerDuration)
	}()
}
