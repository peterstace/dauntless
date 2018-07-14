package main

import (
	"fmt"
	"io"
	"regexp"
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
	msgSetAt            time.Time
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

const msgLingerDuration = 5 * time.Second

func (a *app) Initialise() {
	log.Info("***************** Initialising log viewer ******************")
	a.reactor.SetPostHook(func() {
		// Round cycle to nearest 10 to prevent flapping.
		a.model.cycle = a.reactor.GetCycle() / 10 * 10

		// Check if new message was set, if so prep an event to remove it after
		// the linger duration.
		if a.msgSetAt != a.model.msgSetAt {
			go func() {
				time.Sleep(msgLingerDuration)
				a.reactor.Enque(func() {}, "linger complete")
			}()
		}
		a.msgSetAt = a.model.msgSetAt

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
	} else {
		log.Info("Key press was unhandled: %v", k)
	}
}

// TODO: This whole thing can be part of the model.
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
				a.model.searchEntered(a.model.cmd.Text)
			case ColourCommand:
				a.model.colourEntered(a.model.cmd.Text)
			case SeekCommand:
				if err := a.model.seekEntered(a.model.cmd.Text); err != nil {
					a.reactor.Stop(err)
					return
				}
			case BisectCommand:
				if err := a.model.bisectEntered(a.model.cmd.Text); err != nil {
					a.reactor.Stop(err)
					return
				}
			case QuitCommand:
				a.quitEntered(a.model.cmd.Text)
			case FilterCommand:
				a.model.filterEntered(a.model.cmd.Text)
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

func (a *app) quitEntered(cmd string) {
	switch cmd {
	case "y":
		a.reactor.Stop(nil)
	case "n":
		return
	default:
		a.model.setMessage(fmt.Sprintf("invalid quit response (should be y/n): %v", cmd))
	}
}

func (a *app) discardBufferedInputAndRepaint() {
	log.Info("Discarding buffered input and repainting screen.")
	a.model.fwd = nil
	a.model.bck = nil
	a.model.minLoadOffset = -1
	a.model.maxLoadOffset = -1

	go func() {
		offset, err := FindReloadOffset(a.model.content, a.model.currentOffset)
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

	if lines := a.model.needsLoadingForward(); lines != 0 {
		a.loadForward(lines)
	} else if lines := a.model.needsLoadingBackward(); lines != 0 {
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

func (a *app) loadForward(amount int) {
	offset := a.model.maxLoadOffset
	if offset == -1 {
		offset = a.model.currentOffset
	}
	log.Debug("Loading forward: offset=%d amount=%d", offset, amount)

	a.fillingScreenBuffer = true
	go func() {
		contigLines, err := LoadFwd(a.model.content, offset, amount)
		a.reactor.Enque(func() {
			if err != nil {
				log.Warn("Error loading forward: %v", err)
				a.reactor.Stop(err)
				return
			}
			log.Debug("Got fwd lines: numLines=%d initialFwd=%d initialBck=%d", len(contigLines.lines), len(a.model.fwd), len(a.model.bck))
			if a.model.maxLoadOffset != -1 && a.model.maxLoadOffset != contigLines.minOffset {
				log.Debug("offsets didn't match: modelMaxLoadOffset=%d loadedMinOffset=%d", a.model.maxLoadOffset, contigLines.minOffset)
				return
			}
			offset := contigLines.minOffset
			for _, ln := range contigLines.lines {
				if a.model.filterRegex == nil || a.model.filterRegex.MatchString(ln) {
					a.model.fwd = append(a.model.fwd, line{offset: offset, data: ln})
				}
				offset += len(ln)
			}
			assert(contigLines.maxOffset == offset)
			a.model.maxLoadOffset = offset
			log.Debug("After adding to data structure: fwd=%d bck=%d", len(a.model.fwd), len(a.model.bck))
			a.fillingScreenBuffer = false
		}, "load forward")
	}()
}

func (a *app) loadBackward(amount int) {
	offset := a.model.minLoadOffset
	if offset == -1 {
		offset = a.model.currentOffset
	}
	log.Debug("Loading backward: offset=%d amount=%d", offset, amount)

	a.fillingScreenBuffer = true
	go func() {
		contigLines, err := LoadBck(a.model.content, offset, amount)
		a.reactor.Enque(func() {
			if err != nil {
				log.Warn("Error loading backward: %v", err)
				a.reactor.Stop(err)
				return
			}
			log.Debug("Got bck lines: numLines=%d initialFwd=%d initialBck=%d", len(contigLines.lines), len(a.model.fwd), len(a.model.bck))
			if a.model.minLoadOffset != -1 && a.model.minLoadOffset != contigLines.maxOffset {
				log.Debug("offsets didn't match: modelMinLoadOffset=%d loadedMaxOffset=%d", a.model.minLoadOffset, contigLines.maxOffset)
				return
			}
			offset := contigLines.maxOffset
			for _, ln := range contigLines.lines {
				offset -= len(ln)
				if a.model.filterRegex == nil || a.model.filterRegex.MatchString(ln) {
					a.model.bck = append(a.model.bck, line{offset: offset, data: ln})
				}
			}
			assert(contigLines.minOffset == offset)
			a.model.minLoadOffset = offset
			log.Debug("After adding to data structure: fwd=%d bck=%d", len(a.model.fwd), len(a.model.bck))
			a.fillingScreenBuffer = false
		}, "load backward")
	}()
}

func (a *app) TermSize(rows, cols int, forceRefresh bool) {
	a.forceRefresh = forceRefresh
	a.model.rows = rows
	a.model.cols = cols
	log.Info("Term size: rows=%d cols=%d", rows, cols)
}

func (a *app) FileSize(size int) {
	a.model.FileSize(size)
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

func (a *app) jumpToMatch(reverse bool) {
	re := a.model.currentRE()
	if re == nil {
		msg := "no regex to jump to"
		log.Info(msg)
		a.model.setMessage(msg)
		return
	}

	if len(a.model.fwd) == 0 {
		log.Warn("Cannot search for next match: current line is not loaded.")
		return
	}

	var start int
	if reverse {
		start = a.model.currentOffset
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
					a.model.setMessage(msg)
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
