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
