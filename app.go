package main

import (
	"io"
	"os"
)

type App interface {
	Initialise()
	KeyPress(byte)
	TermSize(rows, cols int, err error)
}

type line struct {
	offset int
	data   string
}

type app struct {
	reactor  Reactor
	filename string
	log      Logger

	rows, cols int

	// Invariants:
	//  1) If fwd is populated, then offset will match the first line.
	//  2) Fwd and bck contain consecutive lines.
	offset int
	fwd    []line
	bck    []line

	fileSize int

	refreshInProgress bool
	refreshPending    bool
	screenBuffer      []byte
}

func NewApp(reactor Reactor, filename string, logger Logger) App {
	return &app{
		reactor:  reactor,
		filename: filename,
		log:      logger,
		rows:     -1,
		cols:     -1,
	}
}

func (a *app) Initialise() {
	a.log.Info("***************** Initialising log viewer ******************")
}

func (a *app) KeyPress(b byte) {

	a.log.Info("Key press: %c", b)

	switch b {

	case 'j':

		if len(a.fwd) == 0 {
			a.log.Warn("Cannot move down: current line not loaded.")
			return
		}

		ln := a.fwd[0]
		assert(ln.offset == a.offset)
		newOffset := ln.offset + len(ln.data)

		if newOffset == a.fileSize {
			a.log.Info("Cannot move down: reached EOF.")
			return
		}

		assert(len(a.fwd) == 1 || newOffset == a.fwd[1].offset)
		a.offset = newOffset
		a.fwd = a.fwd[1:]
		a.bck = append([]line{ln}, a.bck...)
		a.log.Info("Moved down: newOffset=%d", a.offset)
		a.refresh()

	case 'k':

		if a.offset == 0 {
			a.log.Info("Cannot move back: at start of file.")
			return
		}

		if len(a.bck) == 0 {
			a.log.Warn("Cannot move back: previous line not loaded.")
			return
		}

		ln := a.bck[0]
		assert(ln.offset+len(ln.data) == a.offset)
		a.offset = ln.offset
		a.fwd = append([]line{ln}, a.fwd...)
		a.bck = a.bck[1:]
		a.log.Info("Moved down: newOffset=%d", a.offset)
		a.refresh()

	case 'r':

		a.log.Info("Repainting screen")
		a.refresh()

	case 'R':

		a.log.Info("Discarding buffered input and repainting screen")
		a.fwd = nil
		a.bck = nil
		a.refresh()

	case 'g':

		a.log.Info("Jumping to start of file")
		if a.offset == 0 {
			return
		}
		haveOffsetZero := false
		for _, ln := range a.bck {
			if ln.offset == 0 {
				haveOffsetZero = true
			}
		}
		if haveOffsetZero {
			for a.offset != 0 {
				// TODO: Bad performance, but will be okay once a linked list is used.
				ln := a.bck[0]
				a.fwd = append([]line{ln}, a.fwd...)
				a.bck = a.bck[1:]
				a.offset = ln.offset
			}
		} else {
			a.fwd = nil
			a.bck = nil
			a.offset = 0
		}
		a.refresh()

	default:
		a.log.Info("Unhandled key press: %d", b)
	}
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

	if a.refreshInProgress {
		a.log.Info("Refresh requested but one already in progress")
		a.refreshPending = true
		return
	}

	if a.rows < 0 || a.cols < 0 {
		a.log.Warn("Can't refresh, don't know term size yet")
		return
	}

	a.log.Info("Refreshing")
	a.refreshInProgress = true

	if len(a.screenBuffer) != a.rows*a.cols {
		a.screenBuffer = make([]byte, a.rows*a.cols)
	}

	a.renderScreen(a.screenBuffer, a.cols)

	a.log.Info("Writing to screen")
	// TODO: Maybe it's a good idea to wait a little while between writing to
	// the screen each time? To give it some time to 'settle'.
	go func() {
		WriteToTerm(a.screenBuffer)
		a.reactor.Enque(a.notifyRefreshComplete)
	}()
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

func (a *app) notifyRefreshComplete() {
	a.log.Info("Refresh complete")
	a.refreshInProgress = false
	if a.refreshPending {
		a.refreshPending = false
		a.log.Info("Executing pending refresh")
		a.refresh()
	}
}

func (a *app) renderScreen(buf []byte, cols int) {

	a.log.Info("Rendering screen")

	for i := range buf {
		buf[i] = ' '
	}

	assert(len(a.fwd) == 0 || a.fwd[0].offset == a.offset)
	offset := a.offset
	for row := 0; row < len(buf)/cols; row++ {
		if row < len(a.fwd) {
			col := 0
			for i := 0; col+1 < cols && i < len(a.fwd[row].data); i++ {
				col += writeByte(buf[row*cols+col:(row+1)*cols], a.fwd[row].data[i], col)
			}
		} else if len(a.fwd) != 0 && a.fwd[len(a.fwd)-1].offset+len(a.fwd[len(a.fwd)-1].data) >= a.fileSize {
			// Reached end of file.
			assert(a.fwd[len(a.fwd)-1].offset+len(a.fwd[len(a.fwd)-1].data) == a.fileSize) // Assert that it's actually equal.
			break
		} else {
			a.loadData(offset, defaultLoadAmount)
			buildLoadingScreen(buf, cols)
			break
		}
		offset += len(a.fwd[row].data)
	}
}

const defaultLoadAmount = 64

func (a *app) loadData(loadFrom int, amount int) {
	buf := make([]byte, amount)
	var fileInfo os.FileInfo
	var n int
	go func() {

		f, err := os.Open(a.filename)
		if err != nil {
			a.log.Warn("Could not open file: filename=%q reason=%q", a.filename, f)
			a.reactor.Stop()
			return
		}
		n, err = f.ReadAt(buf, int64(loadFrom))
		if err != nil && err != io.EOF {
			a.log.Warn("Could not read file: filename=%q offset=%d reason=%q", a.filename, loadFrom, err)
			a.reactor.Stop()
			return
		}
		fileInfo, err = f.Stat()
		if err != nil {
			a.log.Warn("Could not stat file: filename=%q reason=%q", a.filename, f)
			a.reactor.Stop()
			return
		}

		a.reactor.Enque(func() {
			a.log.Info("Data loaded: From=%d To=%d Len=%d", loadFrom, loadFrom+n, n)
			newFileSize := int(fileInfo.Size())
			if newFileSize != a.fileSize {
				a.log.Info("File size changed: oldSize=%d newSize=%d", a.fileSize, newFileSize)
				a.fileSize = newFileSize
			}
			a.fileSize = newFileSize

			offset := loadFrom
			containedLine := false
			for _, data := range extractLines(loadFrom, buf[:n]) {
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
			} else if reachedEndOfFile := loadFrom+n == a.fileSize; !reachedEndOfFile {
				a.log.Warn("Data loaded didn't contain at least one complete line: retrying with double amount.")
				a.loadData(loadFrom, amount*2)
			} else {
				a.log.Warn("Data loaded didn't contain at least one complete line: reached EOF")
			}
		})
	}()
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
