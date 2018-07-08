package main

import (
	"regexp"
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
