package main

import (
	"io"
	"regexp"
	"time"
)

type App interface {
	Initialise()
	KeyPress(Key)
	TermSize(rows, cols int)
	FileSize(int)
	Interrupt()
	ForceRefresh()
}

type CommandHandler interface {
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

type app struct {
	reactor Reactor
	log     Logger

	screen Screen

	fillingScreenBuffer bool

	forceRefresh bool

	model Model
}

func NewApp(reactor Reactor, filename string, logger Logger, screen Screen, config Config) App {
	return &app{
		reactor: reactor,
		log:     logger,
		screen:  screen,
		model:   Model{config: config, filename: filename, commandReader: new(commandReader)},
	}
}

func (a *app) ForceRefresh() {
	a.forceRefresh = true
}

func (a *app) Initialise() {
	a.log.Info("***************** Initialising log viewer ******************")
	a.reactor.SetPostHook(func() {
		a.fillScreenBuffer()
		a.refresh()
	})
}

func (a *app) Interrupt() {
	a.log.Info("Caught interrupt.")
	if a.model.commandReader.Enabled() {
		a.model.commandReader.Clear()
	} else {
		a.startQuitCommand()
	}
}

func (a *app) KeyPress(k Key) {

	a.log.Info("Key press: %s", k)

	if a.model.commandReader.Enabled() {
		a.model.commandReader.KeyPress(k, a)
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
	for i := 0; i < a.model.rows/2; i++ {
		a.moveDown()
	}
}

func (a *app) moveUpByHalfScreen() {
	for i := 0; i < a.model.rows/2; i++ {
		a.moveUp()
	}
}

func (a *app) discardBufferedInputAndRepaint() {
	a.log.Info("Discarding buffered input and repainting screen.")
	a.model.fwd = nil
	a.model.bck = nil

	go func() {
		offset, err := FindReloadOffset(a.model.filename, a.model.offset)
		a.reactor.Enque(func() {
			if err != nil {
				a.log.Warn("Could not find reload offset: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.moveToOffset(offset)
		})
	}()
}

func (a *app) moveDown() {
	a.log.Info("Moving down.")
	if len(a.model.fwd) < 2 {
		a.log.Warn("Cannot move down: reason=\"not enough lines loaded\" linesLoaded=%d", len(a.model.fwd))
		return
	}
	a.moveToOffset(a.model.fwd[1].offset)
}

func (a *app) moveUp() {

	a.log.Info("Moving up.")

	if a.model.offset == 0 {
		a.log.Info("Cannot move back: at start of file.")
		return
	}

	if len(a.model.bck) == 0 {
		a.log.Warn("Cannot move back: previous line not loaded.")
		return
	}

	a.moveToOffset(a.model.bck[0].offset)
}

func (a *app) moveTop() {
	a.log.Info("Jumping to start of file.")
	a.moveToOffset(0)
}

func (a *app) moveBottom() {

	a.log.Info("Jumping to bottom of file.")

	go func() {
		offset, err := FindJumpToBottomOffset(a.model.filename)
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
	a.log.Info("Moving to offset: currentOffset=%d newOffset=%d", a.model.offset, offset)

	assert(offset >= 0)

	if a.model.offset == offset {
		a.log.Info("Already at target offset.")
	} else if offset < a.model.offset {
		a.moveUpToOffset(offset)
	} else {
		a.moveDownToOffset(offset)
	}
}

func (a *app) moveUpToOffset(offset int) {

	a.log.Info("Moving up to offset: currentOffset=%d newOffset=%d", a.model.offset, offset)

	haveTargetLoaded := false
	for _, ln := range a.model.bck {
		if ln.offset == offset {
			haveTargetLoaded = true
			break
		}
	}
	if haveTargetLoaded {
		for a.model.offset != offset {
			ln := a.model.bck[0]
			a.model.fwd = append([]line{ln}, a.model.fwd...)
			a.model.bck = a.model.bck[1:]
			a.model.offset = ln.offset
		}
	} else {
		a.model.fwd = nil
		a.model.bck = nil
		a.model.offset = offset
	}
}

func (a *app) moveDownToOffset(offset int) {

	a.log.Info("Moving down to offset: currentOffset=%d newOffset=%d", a.model.offset, offset)

	haveTargetLoaded := false
	for _, ln := range a.model.fwd {
		if ln.offset == offset {
			haveTargetLoaded = true
			break
		}
	}
	if haveTargetLoaded {
		for a.model.offset != offset {
			ln := a.model.fwd[0]
			a.model.fwd = a.model.fwd[1:]
			a.model.bck = append([]line{ln}, a.model.bck...)
			a.model.offset = ln.offset + len(ln.data)
		}
	} else {
		a.model.fwd = nil
		a.model.bck = nil
		a.model.offset = offset
	}
}

func (a *app) CommandFailed(err error) {
	a.log.Warn("Command failed: %v", err)
	a.setMessage(err.Error())
}

func (a *app) startSearchCommand() {
	a.model.commandReader.SetMode(search{})
	a.model.msg = ""
	a.log.Info("Accepting search command.")
}

func (a *app) SearchCommandEntered(re *regexp.Regexp) {
	a.model.tmpRegex = re
}

func (a *app) startColourCommand() {
	if a.currentRE() == nil {
		msg := "cannot select regex color: no active regex"
		a.log.Warn(msg)
		a.setMessage(msg)
		return
	}
	a.model.commandReader.SetMode(colour{})
	a.model.msg = ""
	a.log.Info("Accepting colour command.")
}

func (a *app) ColourCommandEntered(style Style) {
	if a.model.tmpRegex != nil {
		a.model.regexes = append([]regex{{style, a.model.tmpRegex}}, a.model.regexes...)
		a.model.tmpRegex = nil
	} else if len(a.model.regexes) > 0 {
		a.model.regexes[0].style = style
	} else {
		// Should not have been allowed to start the colour command.
		assert(false)
	}
}

func (a *app) startSeekCommand() {
	a.model.commandReader.SetMode(seek{})
	a.model.msg = ""
	a.log.Info("Accepting seek command.")
}

func (a *app) SeekCommandEntered(pct float64) {
	go func() {
		offset, err := FindSeekOffset(a.model.filename, pct)
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
	a.model.commandReader.SetMode(bisect{})
	a.model.msg = ""
	a.log.Info("Accepting bisect command.")
}

func (a *app) BisectCommandEntered(target string) {
	a.log.Info("Bisect command entered: %q", target)
	go func() {
		offset, err := Bisect(a.model.filename, target, a.model.config.BisectMask)
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
	a.model.commandReader.SetMode(quit{})
	a.model.msg = ""
	a.log.Info("Accepting quit command.")
}

func (a *app) QuitCommandEntered(quit bool) {
	if quit {
		a.reactor.Stop(nil)
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

	if len(a.model.fwd) == 0 {
		a.log.Warn("Cannot search for next match: current line is not loaded.")
		return
	}
	startOffset := a.model.fwd[0].nextOffset()

	a.log.Info("Searching for next regexp match: regexp=%q", re)

	go func() {
		offset, err := FindNextMatch(a.model.filename, startOffset, re)
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

	endOffset := a.model.offset

	a.log.Info("Searching for previous regexp match: regexp=%q", re)

	go func() {
		offset, err := FindPrevMatch(a.model.filename, endOffset, re)
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
	if a.model.lineWrapMode {
		a.log.Info("Toggling out of line wrap mode.")
	} else {
		a.log.Info("Toggling into line wrap mode.")
	}
	a.model.lineWrapMode = !a.model.lineWrapMode
	a.model.xPosition = 0
}

func (a *app) cycleRegexp() {

	if len(a.model.regexes) == 0 {
		msg := "no regexes to cycle between"
		a.log.Warn(msg)
		a.setMessage(msg)
		return
	}

	a.model.tmpRegex = nil // Any temp re gets discarded.
	a.model.regexes = append(a.model.regexes[1:], a.model.regexes[0])
}

func (a *app) deleteRegexp() {
	if a.model.tmpRegex != nil {
		a.model.tmpRegex = nil
	} else if len(a.model.regexes) > 0 {
		a.model.regexes = a.model.regexes[1:]
	} else {
		msg := "no regexes to delete"
		a.log.Warn(msg)
		a.setMessage(msg)
	}
}

func (a *app) reduceXPosition() {
	a.changeXPosition(max(0, a.model.xPosition-a.model.cols/4))
}

func (a *app) increaseXPosition() {
	a.changeXPosition(max(0, a.model.xPosition+a.model.cols/4))
}

func (a *app) changeXPosition(newPosition int) {
	a.log.Info("Changing x position: old=%v new=%v", a.model.xPosition, newPosition)
	if a.model.xPosition != newPosition {
		a.model.xPosition = newPosition
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

	a.log.Info("Filling screen buffer, has initial state: fwd=%d bck=%d", len(a.model.fwd), len(a.model.bck))

	if lines := a.needsLoadingForward(); lines != 0 {
		a.loadForward(lines)
	} else if lines := a.needsLoadingBackward(); lines != 0 {
		a.loadBackward(lines)
	} else {
		a.log.Info("Screen buffer didn't need filling.")
	}

	// Prune buffers.
	neededFwd := min(len(a.model.fwd), a.model.rows*forwardUnloadFactor)
	a.model.fwd = a.model.fwd[:neededFwd]
	neededBck := min(len(a.model.bck), a.model.rows*backUnloadFactor)
	a.model.bck = a.model.bck[:neededBck]
}

func (a *app) needsLoadingForward() int {
	if a.model.fileSize == 0 {
		return 0
	}
	if len(a.model.fwd) >= a.model.rows*forwardLoadFactor {
		return 0
	}
	if len(a.model.fwd) > 0 {
		lastLine := a.model.fwd[len(a.model.fwd)-1]
		if lastLine.offset+len(lastLine.data) >= a.model.fileSize {
			return 0
		}
	}
	return a.model.rows*forwardLoadFactor - len(a.model.fwd)
}

func (a *app) needsLoadingBackward() int {
	if a.model.offset == 0 {
		return 0
	}
	if len(a.model.bck) >= a.model.rows*backLoadFactor {
		return 0
	}
	if len(a.model.bck) > 0 {
		lastLine := a.model.bck[len(a.model.bck)-1]
		if lastLine.offset == 0 {
			return 0
		}
	}
	return a.model.rows*backLoadFactor - len(a.model.bck)
}

func (a *app) loadForward(amount int) {

	offset := a.model.offset
	if len(a.model.fwd) > 0 {
		offset = a.model.fwd[len(a.model.fwd)-1].nextOffset()
	}
	a.log.Debug("Loading forward: offset=%d amount=%d", offset, amount)

	a.fillingScreenBuffer = true
	go func() {
		lines, err := LoadFwd(a.model.filename, offset, amount)
		a.reactor.Enque(func() {
			if err != nil {
				a.log.Warn("Error loading forward: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.log.Debug("Got fwd lines: numLines=%d initialFwd=%d initialBck=%d", len(lines), len(a.model.fwd), len(a.model.bck))
			for _, data := range lines {
				if (len(a.model.fwd) == 0 && offset == a.model.offset) ||
					(len(a.model.fwd) > 0 && a.model.fwd[len(a.model.fwd)-1].nextOffset() == offset) {
					a.model.fwd = append(a.model.fwd, line{offset, data})
				}
				offset += len(data)
			}
			a.log.Debug("After adding to data structure: fwd=%d bck=%d", len(a.model.fwd), len(a.model.bck))
			a.fillingScreenBuffer = false
		})
	}()
}

func (a *app) loadBackward(amount int) {

	offset := a.model.offset
	if len(a.model.bck) > 0 {
		offset = a.model.bck[len(a.model.bck)-1].offset
	}
	a.log.Debug("Loading backward: offset=%d amount=%d", offset, amount)

	a.fillingScreenBuffer = true
	go func() {
		lines, err := LoadBck(a.model.filename, offset, amount)
		a.reactor.Enque(func() {
			if err != nil {
				a.log.Warn("Error loading backward: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.log.Debug("Got bck lines: numLines=%d initialFwd=%d initialBck=%d", len(lines), len(a.model.fwd), len(a.model.bck))
			for _, data := range lines {
				if (len(a.model.bck) == 0 && offset == a.model.offset) ||
					(len(a.model.bck) > 0 && a.model.bck[len(a.model.bck)-1].offset == offset) {
					a.model.bck = append(a.model.bck, line{offset - len(data), data})
				}
				offset -= len(data)
			}
			a.log.Debug("After adding to data structure: fwd=%d bck=%d", len(a.model.fwd), len(a.model.bck))
			a.fillingScreenBuffer = false
		})
	}()
}

func (a *app) TermSize(rows, cols int) {
	if a.model.rows != rows || a.model.cols != cols {
		a.model.rows = rows
		a.model.cols = cols
		a.log.Info("Term size: rows=%d cols=%d", rows, cols)
	}
}

func (a *app) FileSize(size int) {
	oldSize := a.model.fileSize
	if size != oldSize {
		a.model.fileSize = size
		a.log.Info("File size changed: old=%d new=%d", oldSize, size)
		if len(a.model.fwd) > 0 {
			lastLine := a.model.fwd[len(a.model.fwd)-1].data
			if lastLine[len(lastLine)-1] != '\n' {
				a.model.fwd = a.model.fwd[:len(a.model.fwd)-1]
			}
		}
	}
}

func (a *app) refresh() {
	a.log.Info("Refreshing")
	if a.model.cols == 0 || a.model.rows == 0 {
		a.log.Info("Aborting refresh: rows=%d cols=%d", a.model.rows, a.model.cols)
		return
	}
	a.renderScreen()
}

func (a *app) renderScreen() {
	state := CreateView(&a.model)
	a.screen.Write(state, a.forceRefresh)
	a.forceRefresh = false
}

func (a *app) currentRE() *regexp.Regexp {
	re := a.model.tmpRegex
	if re == nil && len(a.model.regexes) > 0 {
		re = a.model.regexes[0].re
	}
	return re
}

const msgLingerDuration = 5 * time.Second

func (a *app) setMessage(msg string) {
	a.log.Info("Setting message: %q", msg)
	a.model.msg = msg
	a.model.msgSetAt = time.Now()
	go func() {
		time.Sleep(msgLingerDuration)
	}()
}
