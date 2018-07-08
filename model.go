package main

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"time"
)

type Model struct {
	config Config

	content  Content
	filename string

	rows, cols int

	// Invariants:
	//  1) If fwd is populated, then offset will match the first line.
	//  2) Fwd and bck contain consecutive lines.
	offset int
	fwd    []line
	bck    []line

	fileSize int

	tmpRegex *regexp.Regexp
	regexes  []regex

	lineWrapMode bool
	xPosition    int

	msg      string
	msgSetAt time.Time

	cmd Command

	debug bool
	cycle int

	longFileOpInProgress bool
	cancelLongFileOp     Cancellable

	history    map[CommandMode][]string // most recent is first in list
	historyIdx int                      // -1 when history not used

	showHelp bool
}

type Command struct {
	Mode CommandMode
	Text string
	Pos  int
}

type CommandMode int

const (
	NoCommand CommandMode = iota
	SearchCommand
	ColourCommand
	SeekCommand
	BisectCommand
	QuitCommand
)

type regex struct {
	style Style
	re    *regexp.Regexp
}

type line struct {
	offset int
	data   string
}

func (l line) nextOffset() int {
	return l.offset + len(l.data)
}

func (m *Model) StartCommandMode(mode CommandMode) {
	m.cmd.Mode = mode
	m.msg = ""
	m.historyIdx = -1
}

func (m *Model) ExitCommandMode() {
	// Remove from history (if exists), then add back in at start.
	mode := m.cmd.Mode
	txt := m.cmd.Text
	for i, h := range m.history[mode] {
		if h == txt {
			m.history[mode] = append(
				m.history[mode][:i],
				m.history[mode][i+1:]...,
			)
			break
		}
	}
	m.history[mode] = append(
		[]string{txt},
		m.history[mode]...,
	)

	// Reset command.
	m.cmd.Mode = NoCommand
	m.cmd.Text = ""
	m.cmd.Pos = 0
}

func (m *Model) BackInHistory() {
	hist := m.history[m.cmd.Mode]
	if len(hist) == 0 || m.historyIdx+1 >= len(hist) {
		return
	}
	m.historyIdx++
	m.cmd.Text = hist[m.historyIdx]
	m.cmd.Pos = len(m.cmd.Text)
}

func (m *Model) ForwardInHistory() {
	hist := m.history[m.cmd.Mode]
	if len(hist) == 0 || m.historyIdx == 0 {
		return
	}
	m.historyIdx--
	m.cmd.Text = hist[m.historyIdx]
	m.cmd.Pos = len(m.cmd.Text)
}

func (m *Model) Interrupt() {
	log.Info("Caught interrupt.")
	if m.cmd.Mode != NoCommand {
		m.cmd.Mode = NoCommand
		m.cmd.Text = ""
		m.cmd.Pos = 0
	} else if m.longFileOpInProgress {
		m.cancelLongFileOp.Cancel()
		m.longFileOpInProgress = false
	} else {
		m.StartCommandMode(QuitCommand)
	}
}

func (m *Model) moveToOffset(offset int) {
	log.Info("Moving to offset: currentOffset=%d newOffset=%d", m.offset, offset)
	assert(offset >= 0)
	if m.offset == offset {
		log.Info("Already at target offset.")
		return
	}

	ahead, aback := &m.fwd, &m.bck
	if offset < m.offset {
		ahead, aback = aback, ahead
	}

	for _, ln := range *ahead {
		if ln.offset == offset {
			for m.offset != offset {
				l := (*ahead)[0]
				*ahead = (*ahead)[1:]
				*aback = append([]line{l}, *aback...)
				m.offset = l.offset
				if ahead == &m.fwd {
					m.offset += len(l.data)
				}
			}
			return
		}
	}
	m.fwd = nil
	m.bck = nil
	m.offset = offset
}

func (m *Model) moveDown() {
	log.Info("Moving down.")
	if len(m.fwd) < 2 {
		log.Warn("Cannot move down: reason=\"not enough lines loaded\" linesLoaded=%d", len(m.fwd))
		return
	}
	m.moveToOffset(m.fwd[1].offset)
}

func (m *Model) moveUp() {
	log.Info("Moving up.")
	if m.offset == 0 {
		log.Info("Cannot move back: at start of file.")
		return
	}
	if len(m.bck) == 0 {
		log.Warn("Cannot move back: previous line not loaded.")
		return
	}
	m.moveToOffset(m.bck[0].offset)
}

func (m *Model) moveTop() {
	log.Info("Jumping to start of file.")
	m.moveToOffset(0)
}

func (m *Model) moveDownByHalfScreen() {
	for i := 0; i < m.rows/2; i++ {
		m.moveDown()
	}
}

func (m *Model) moveUpByHalfScreen() {
	for i := 0; i < m.rows/2; i++ {
		m.moveUp()
	}
}

func (m *Model) toggleLineWrapMode() {
	if m.lineWrapMode {
		log.Info("Toggling out of line wrap mode.")
	} else {
		log.Info("Toggling into line wrap mode.")
	}
	m.lineWrapMode = !m.lineWrapMode
	m.xPosition = 0
}

func (m *Model) currentRE() *regexp.Regexp {
	re := m.tmpRegex
	if re == nil && len(m.regexes) > 0 {
		re = m.regexes[0].re
	}
	return re
}

func (m *Model) setMessage(msg string) {
	m.msg = msg
	m.msgSetAt = time.Now()
}

func (m *Model) reduceXPosition() {
	m.changeXPosition(max(0, m.xPosition-m.cols/4))
}

func (m *Model) increaseXPosition() {
	m.changeXPosition(max(0, m.xPosition+m.cols/4))
}

func (m *Model) changeXPosition(newPosition int) {
	log.Info("Changing x position: old=%v new=%v", m.xPosition, newPosition)
	if m.xPosition != newPosition {
		m.xPosition = newPosition
	}
}

func (m *Model) startColourCommand() {
	if m.currentRE() == nil {
		msg := "cannot select regex color: no active regex"
		m.setMessage(msg)
		return
	}
	m.StartCommandMode(ColourCommand)
	log.Info("Accepting colour command.")
}

func (m *Model) cycleRegexp(forward bool) {
	if len(m.regexes) == 0 {
		msg := "no regexes to cycle between"
		log.Warn(msg)
		m.setMessage(msg)
		return
	}

	m.tmpRegex = nil // Any temp re gets discarded.
	if forward {
		m.regexes = append(m.regexes[1:], m.regexes[0])
	} else {
		m.regexes = append(
			[]regex{m.regexes[len(m.regexes)-1]},
			m.regexes[:len(m.regexes)-1]...,
		)
	}
}

func (m *Model) deleteRegexp() {
	if m.tmpRegex != nil {
		m.tmpRegex = nil
	} else if len(m.regexes) > 0 {
		m.regexes = m.regexes[1:]
	} else {
		msg := "no regexes to delete"
		log.Warn(msg)
		m.setMessage(msg)
	}
}

func (m *Model) FileSize(size int) {
	oldSize := m.fileSize
	log.Info("File size changed: old=%d new=%d", oldSize, size)
	m.fileSize = size
	if len(m.fwd) == 0 {
		return
	}
	if m.fwd[len(m.fwd)-1].nextOffset() == oldSize {
		m.fwd = m.fwd[:len(m.fwd)-1]
	}
}

func (m *Model) searchEntered(cmd string) {
	re, err := regexp.Compile(cmd)
	if err != nil {
		m.setMessage(err.Error())
		return
	}
	m.tmpRegex = re
}

func (m *Model) colourEntered(cmd string) {
	err := fmt.Errorf("colour code must be in format [0-8][0-8]: %v", cmd)
	if len(cmd) != 2 {
		m.setMessage(err.Error())
		return
	}
	fg := cmd[0]
	bg := cmd[1]
	if fg < '0' || fg > '8' || bg < '0' || bg > '8' {
		m.setMessage(err.Error())
		return
	}

	style := MixStyle(styles[fg-'0'], styles[bg-'0'])
	if m.tmpRegex != nil {
		m.regexes = append([]regex{{style, m.tmpRegex}}, m.regexes...)
		m.tmpRegex = nil
	} else if len(m.regexes) > 0 {
		m.regexes[0].style = style
	} else {
		// Should not have been allowed to start the colour command.
		assert(false)
	}
}

func (m *Model) seekEntered(cmd string) error {
	seekPct, err := strconv.ParseFloat(cmd, 64)
	if err != nil {
		m.setMessage(err.Error())
		return nil
	}
	if seekPct < 0 || seekPct > 100 {
		m.setMessage(fmt.Sprintf("seek percentage out of range [0, 100]: %v", seekPct))
		return nil
	}

	offset, err := FindSeekOffset(m.content, seekPct)
	if err != nil {
		log.Warn("Could to find start of line at offset: %v", err)
		return err
	}

	m.moveToOffset(offset)
	return nil
}

func (m *Model) bisectEntered(cmd string) error {
	sz, err := m.content.Size()
	if err != nil {
		return err
	}

	var start int
	end := int(sz - 1)

	var i int
	for {
		i++
		if i == 1000 {
			m.setMessage("could not find bisect target after 1000 iterations")
			return nil
		}

		offset := start + rand.Intn(end-start+1)
		line, offset, err := lineAt(m.content, offset)
		if err != nil {
			return err
		}
		if start+len(line) >= end {
			break
		}
		if m.config.BisectMask.MatchString(transform(line)) {
			if cmd < string(line) {
				end = offset
			} else {
				start = offset
			}
		}
	}
	m.moveToOffset(start)
	return nil
}

func (m *Model) needsLoadingForward() int {
	if m.fileSize == 0 {
		return 0
	}
	if len(m.fwd) >= m.rows*forwardLoadFactor {
		return 0
	}
	if len(m.fwd) > 0 {
		lastLine := m.fwd[len(m.fwd)-1]
		if lastLine.offset+len(lastLine.data) >= m.fileSize {
			return 0
		}
	}
	return m.rows*forwardLoadFactor - len(m.fwd)
}

func (m *Model) needsLoadingBackward() int {
	if m.offset == 0 {
		return 0
	}
	if len(m.bck) >= m.rows*backLoadFactor {
		return 0
	}
	if len(m.bck) > 0 {
		lastLine := m.bck[len(m.bck)-1]
		if lastLine.offset == 0 {
			return 0
		}
	}
	return m.rows*backLoadFactor - len(m.bck)
}
