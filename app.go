package main

import (
	"fmt"
	"io"
	"math/rand"
	"regexp"
	"strconv"
	"time"
)

type App interface {
	Initialise()
	KeyPress(Key)
	Interrupt()
	TermSize(rows, cols int, forceRefresh bool)
	FileSize(int)
}

type app struct {
	reactor             Reactor
	screen              Screen
	fillingScreenBuffer bool
	forceRefresh        bool
	model               Model
}

func NewApp(reactor Reactor, content Content, filename string, screen Screen, config Config) App {
	return &app{
		reactor: reactor,
		screen:  screen,
		model: Model{
			config:   config,
			content:  content,
			filename: filename,
			history:  map[CommandMode][]string{},
		},
	}
}

func (a *app) Initialise() {
	log.Info("***************** Initialising log viewer ******************")
	a.reactor.SetPostHook(func() {
		// Round cycle to nearest 10 to prevent flapping.
		a.model.cycle = a.reactor.GetCycle() / 10 * 10
		a.fillScreenBuffer()
		a.refresh()
	})
}

func (a *app) Interrupt() {
	a.model.Interrupt()
}

func (a *app) KeyPress(k Key) {
	if a.model.longFileOpInProgress {
		return
	}
	if a.model.cmd.Mode == NoCommand {
		a.normalModeKeyPress(k)
	} else {
		a.commandModeKeyPress(k)
	}
}

func (a *app) normalModeKeyPress(k Key) {
	assert(a.model.cmd.Mode == NoCommand)
	var ctrl *control
	for i := range controls {
		for j := range controls[i].keys {
			if k == controls[i].keys[j] {
				ctrl = &controls[i]
				break
			}
		}
		if ctrl != nil {
			break
		}
	}
	if ctrl != nil {
		ctrl.action(a)
	}
	log.Info("Key press was unhandled: %v", k)
}

func (a *app) commandModeKeyPress(k Key) {
	assert(a.model.cmd.Mode != NoCommand)
	if len(k) == 1 {
		b := k[0]
		if b >= ' ' && b <= '~' {
			a.model.cmd.Text = a.model.cmd.Text[:a.model.cmd.Pos] + string([]byte{b}) + a.model.cmd.Text[a.model.cmd.Pos:]
			a.model.cmd.Pos++
		} else if b == 127 && len(a.model.cmd.Text) >= 1 {
			a.model.cmd.Text = a.model.cmd.Text[:a.model.cmd.Pos-1] + a.model.cmd.Text[a.model.cmd.Pos:]
			a.model.cmd.Pos--
		} else if b == '\n' {
			switch a.model.cmd.Mode {
			case SearchCommand:
				a.searchEntered(a.model.cmd.Text)
			case ColourCommand:
				a.colourEntered(a.model.cmd.Text)
			case SeekCommand:
				a.seekEntered(a.model.cmd.Text)
			case BisectCommand:
				a.bisectEntered(a.model.cmd.Text)
			case QuitCommand:
				a.quitEntered(a.model.cmd.Text)
			default:
				assert(false)
			}
			a.model.ExitCommandMode()
		}
	} else {
		switch k {
		case LeftArrowKey:
			a.model.cmd.Pos = max(0, a.model.cmd.Pos-1)
		case RightArrowKey:
			a.model.cmd.Pos = min(a.model.cmd.Pos+1, len(a.model.cmd.Text))
		case UpArrowKey:
			a.model.BackInHistory()
		case DownArrowKey:
			a.model.ForwardInHistory()
		case DeleteKey:
			if a.model.cmd.Pos < len(a.model.cmd.Text) {
				a.model.cmd.Text = a.model.cmd.Text[:a.model.cmd.Pos] + a.model.cmd.Text[a.model.cmd.Pos+1:]
			}
		case HomeKey:
			a.model.cmd.Pos = 0
		case EndKey:
			a.model.cmd.Pos = len(a.model.cmd.Text)
		}
	}
}

var styles = [...]Style{Default, Black, Red, Green, Yellow, Blue, Magenta, Cyan, White}

func (a *app) colourEntered(cmd string) {
	err := fmt.Errorf("colour code must be in format [0-8][0-8]: %v", cmd)
	if len(cmd) != 2 {
		a.CommandFailed(err)
		return
	}
	fg := cmd[0]
	bg := cmd[1]
	if fg < '0' || fg > '8' || bg < '0' || bg > '8' {
		a.CommandFailed(err)
		return
	}

	style := MixStyle(styles[fg-'0'], styles[bg-'0'])
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
func (a *app) seekEntered(cmd string) {
	seekPct, err := strconv.ParseFloat(cmd, 64)
	if err != nil {
		a.CommandFailed(err)
		return
	}
	if seekPct < 0 || seekPct > 100 {
		a.CommandFailed(fmt.Errorf("seek percentage out of range [0, 100]: %v", seekPct))
		return
	}

	go func() {
		offset, err := FindSeekOffset(a.model.content, seekPct)
		a.reactor.Enque(func() {
			if err != nil {
				log.Warn("Could to find start of line at offset: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.model.moveToOffset(offset)
		}, "seek entered")
	}()
}

func (a *app) bisectEntered(cmd string) {
	a.model.longFileOpInProgress = true
	a.model.cancelLongFileOp.Reset()
	a.model.msg = ""

	log.Info("bisecting: %s", cmd)
	go a.asyncBisect(cmd)
}

func (a *app) asyncBisect(target string) {
	defer a.reactor.Enque(func() { a.model.longFileOpInProgress = false }, "find match complete")

	sz, err := a.model.content.Size()
	if err != nil {
		a.reactor.Stop(err)
		return
	}

	var start int
	end := int(sz - 1)

	var i int
	for {
		i++
		if i == 1000 {
			a.reactor.Enque(func() {
				a.setMessage("could not find bisect target after 1000 iterations")
			}, "could not find bisect target")
			return
		}

		if a.model.cancelLongFileOp.Cancelled() {
			return
		}

		offset := start + rand.Intn(end-start+1)
		line, offset, err := lineAt(a.model.content, offset)
		if err != nil {
			a.reactor.Stop(err)
			return
		}
		if start+len(line) >= end {
			break
		}
		if a.model.config.BisectMask.MatchString(transform(line)) {
			if target < string(line) {
				end = offset
			} else {
				start = offset
			}
		}
	}

	a.reactor.Enque(func() {
		a.model.moveToOffset(start)
	}, "bisect complete")
}

func (a *app) quitEntered(cmd string) {
	switch cmd {
	case "y":
		a.reactor.Stop(nil)
	case "n":
		return
	default:
		a.CommandFailed(fmt.Errorf("invalid quit response (should be y/n): %v", cmd))
	}
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
	log.Info("Discarding buffered input and repainting screen.")
	a.model.fwd = nil
	a.model.bck = nil

	go func() {
		offset, err := FindReloadOffset(a.model.content, a.model.offset)
		a.reactor.Enque(func() {
			if err != nil {
				log.Warn("Could not find reload offset: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.model.moveToOffset(offset)
		}, "discard buffered input and repaint")
	}()
}

func (a *app) moveDown() {
	log.Info("Moving down.")
	if len(a.model.fwd) < 2 {
		log.Warn("Cannot move down: reason=\"not enough lines loaded\" linesLoaded=%d", len(a.model.fwd))
		return
	}
	a.model.moveToOffset(a.model.fwd[1].offset)
}

func (a *app) moveUp() {

	log.Info("Moving up.")

	if a.model.offset == 0 {
		log.Info("Cannot move back: at start of file.")
		return
	}

	if len(a.model.bck) == 0 {
		log.Warn("Cannot move back: previous line not loaded.")
		return
	}

	a.model.moveToOffset(a.model.bck[0].offset)
}

func (a *app) moveTop() {
	log.Info("Jumping to start of file.")
	a.model.moveToOffset(0)
}

func (a *app) moveBottom() {

	log.Info("Jumping to bottom of file.")

	go func() {
		offset, err := FindJumpToBottomOffset(a.model.content)
		a.reactor.Enque(func() {
			if err != nil {
				log.Warn("Could not find jump-to-bottom offset: %v", err)
				a.reactor.Stop(err)
				return
			}
			a.model.moveToOffset(offset)
		}, "move bottom")
	}()
}

func (a *app) CommandFailed(err error) {
	log.Warn("Command failed: %v", err)
	a.setMessage(err.Error())
}

func currentRE(m *Model) *regexp.Regexp {
	re := m.tmpRegex
	if re == nil && len(m.regexes) > 0 {
		re = m.regexes[0].re
	}
	return re
}

func (a *app) startColourCommand() {
	if currentRE(&a.model) == nil {
		msg := "cannot select regex color: no active regex"
		a.setMessage(msg)
		return
	}
	a.model.StartCommandMode(ColourCommand)
	log.Info("Accepting colour command.")
}

func (a *app) toggleLineWrapMode() {
	if a.model.lineWrapMode {
		log.Info("Toggling out of line wrap mode.")
	} else {
		log.Info("Toggling into line wrap mode.")
	}
	a.model.lineWrapMode = !a.model.lineWrapMode
	a.model.xPosition = 0
}

func (a *app) cycleRegexp(forward bool) {
	if len(a.model.regexes) == 0 {
		msg := "no regexes to cycle between"
		log.Warn(msg)
		a.setMessage(msg)
		return
	}

	a.model.tmpRegex = nil // Any temp re gets discarded.
	if forward {
		a.model.regexes = append(a.model.regexes[1:], a.model.regexes[0])
	} else {
		a.model.regexes = append(
			[]regex{a.model.regexes[len(a.model.regexes)-1]},
			a.model.regexes[:len(a.model.regexes)-1]...,
		)
	}
}

func (a *app) deleteRegexp() {
	if a.model.tmpRegex != nil {
		a.model.tmpRegex = nil
	} else if len(a.model.regexes) > 0 {
		a.model.regexes = a.model.regexes[1:]
	} else {
		msg := "no regexes to delete"
		log.Warn(msg)
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
	log.Info("Changing x position: old=%v new=%v", a.model.xPosition, newPosition)
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
		log.Info("Aborting filling screen buffer, already in progress.")
		return
	}

	log.Info("Filling screen buffer, has initial state: fwd=%d bck=%d", len(a.model.fwd), len(a.model.bck))

	if lines := a.needsLoadingForward(); lines != 0 {
		a.loadForward(lines)
	} else if lines := a.needsLoadingBackward(); lines != 0 {
		a.loadBackward(lines)
	} else {
		log.Info("Screen buffer didn't need filling.")
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
	log.Debug("Loading forward: offset=%d amount=%d", offset, amount)

	a.fillingScreenBuffer = true
	go func() {
		lines, err := LoadFwd(a.model.content, offset, amount)
		a.reactor.Enque(func() {
			if err != nil {
				log.Warn("Error loading forward: %v", err)
				a.reactor.Stop(err)
				return
			}
			log.Debug("Got fwd lines: numLines=%d initialFwd=%d initialBck=%d", len(lines), len(a.model.fwd), len(a.model.bck))
			for _, data := range lines {
				if (len(a.model.fwd) == 0 && offset == a.model.offset) ||
					(len(a.model.fwd) > 0 && a.model.fwd[len(a.model.fwd)-1].nextOffset() == offset) {
					a.model.fwd = append(a.model.fwd, line{offset, data})
				}
				offset += len(data)
			}
			log.Debug("After adding to data structure: fwd=%d bck=%d", len(a.model.fwd), len(a.model.bck))
			a.fillingScreenBuffer = false
		}, "load forward")
	}()
}

func (a *app) loadBackward(amount int) {

	offset := a.model.offset
	if len(a.model.bck) > 0 {
		offset = a.model.bck[len(a.model.bck)-1].offset
	}
	log.Debug("Loading backward: offset=%d amount=%d", offset, amount)

	a.fillingScreenBuffer = true
	go func() {
		lines, err := LoadBck(a.model.content, offset, amount)
		a.reactor.Enque(func() {
			if err != nil {
				log.Warn("Error loading backward: %v", err)
				a.reactor.Stop(err)
				return
			}
			log.Debug("Got bck lines: numLines=%d initialFwd=%d initialBck=%d", len(lines), len(a.model.fwd), len(a.model.bck))
			for _, data := range lines {
				if (len(a.model.bck) == 0 && offset == a.model.offset) ||
					(len(a.model.bck) > 0 && a.model.bck[len(a.model.bck)-1].offset == offset) {
					a.model.bck = append(a.model.bck, line{offset - len(data), data})
				}
				offset -= len(data)
			}
			log.Debug("After adding to data structure: fwd=%d bck=%d", len(a.model.fwd), len(a.model.bck))
			a.fillingScreenBuffer = false
		}, "load backward")
	}()
}

func (a *app) TermSize(rows, cols int, forceRefresh bool) {
	// TODO: Probably don't have to check model cols/rows here since duplicates
	// are already filtered.
	a.forceRefresh = forceRefresh
	if a.model.rows != rows || a.model.cols != cols {
		a.model.rows = rows
		a.model.cols = cols
		log.Info("Term size: rows=%d cols=%d", rows, cols)
	}
}

func (a *app) FileSize(size int) {
	oldSize := a.model.fileSize
	if size != oldSize {
		a.model.fileSize = size
		log.Info("File size changed: old=%d new=%d", oldSize, size)
		if len(a.model.fwd) > 0 {
			lastLine := a.model.fwd[len(a.model.fwd)-1].data
			if lastLine[len(lastLine)-1] != '\n' {
				a.model.fwd = a.model.fwd[:len(a.model.fwd)-1]
			}
		}
	}
}

func (a *app) refresh() {
	log.Info("Refreshing")
	if a.model.cols == 0 || a.model.rows == 0 {
		log.Info("Aborting refresh: rows=%d cols=%d", a.model.rows, a.model.cols)
		return
	}
	a.renderScreen()
}

func (a *app) renderScreen() {
	state := CreateView(&a.model)
	a.screen.Write(state, a.forceRefresh)
	a.forceRefresh = false
}

const msgLingerDuration = 5 * time.Second

func (a *app) setMessage(msg string) {
	log.Info("Setting message: %q", msg)
	a.model.msg = msg
	a.model.msgSetAt = time.Now()
	go func() {
		// Trigger an event after the linger duration to stop the message being
		// drawn.
		time.Sleep(msgLingerDuration)
		a.reactor.Enque(func() {}, "linger complete")
	}()
}

func (a *app) searchEntered(cmd string) {
	re, err := regexp.Compile(cmd)
	if err != nil {
		a.CommandFailed(err)
		return
	}
	a.model.tmpRegex = re
}

func (a *app) jumpToMatch(reverse bool) {
	re := currentRE(&a.model)
	if re == nil {
		msg := "no regex to jump to"
		log.Info(msg)
		a.setMessage(msg)
		return
	}

	if len(a.model.fwd) == 0 {
		log.Warn("Cannot search for next match: current line is not loaded.")
		return
	}

	var start int
	if reverse {
		start = a.model.offset
	} else {
		start = a.model.fwd[0].nextOffset()
	}

	a.model.longFileOpInProgress = true
	a.model.cancelLongFileOp.Reset()
	a.model.msg = ""

	log.Info("Searching for next regexp match: regexp=%q", re)

	go a.asyncFindMatch(start, re, reverse)
}

func (a *app) asyncFindMatch(start int, re *regexp.Regexp, reverse bool) {
	defer a.reactor.Enque(func() { a.model.longFileOpInProgress = false }, "find match complete")

	var lineReader LineReader
	if reverse {
		lineReader = NewBackwardLineReader(a.model.content, start)
	} else {
		lineReader = NewForwardLineReader(a.model.content, start)
	}

	offset := start
	for {
		if a.model.cancelLongFileOp.Cancelled() {
			return
		}
		line, err := lineReader.ReadLine()
		if err != nil {
			if err != io.EOF {
				a.reactor.Stop(fmt.Errorf("Could not read: error=%v", err))
				return
			} else {
				a.reactor.Enque(func() {
					msg := "regex search complete: no match found"
					a.setMessage(msg)
				}, "no match found")
				return
			}
		}
		if reverse {
			offset -= len(line)
		}
		if re.MatchString(transform(line)) {
			break
		}
		if !reverse {
			offset += len(line)
		}
	}

	a.reactor.Enque(func() {
		log.Info("Regexp search completed with match.")
		a.model.moveToOffset(offset)
	}, "match found")
}
