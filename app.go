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
	a.log("***************** Initialising log viewer ******************")
}

func (a *app) KeyPress(b byte) {

	switch b {

	case 'j':
		a.log("Key press: j")
	case 'k':
		a.log("Key press: k")
	default:
		a.log("Unhandled key press: %d", b)
	}
}

func (a *app) TermSize(rows, cols int, err error) {
	if a.rows != rows || a.cols != cols {
		a.rows = rows
		a.cols = cols
		a.log("Term size: rows=%d cols=%d", rows, cols)
		a.refresh()
	}
}

func (a *app) refresh() {

	if a.refreshInProgress {
		a.log("Refresh requested but one already in progress")
		a.refreshPending = true
		return
	}
	a.refreshInProgress = true

	a.log("Refreshing")

	if a.rows < 0 || a.cols < 0 {
		a.log("Can't refresh, don't know term size yet")
		return
	}

	if len(a.screenBuffer) != a.rows*a.cols {
		a.screenBuffer = make([]byte, a.rows*a.cols)
	}

	a.renderScreen(a.screenBuffer, a.cols)

	a.log("Writing to screen")
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
	a.log("Refresh complete")
	a.refreshInProgress = false
	if a.refreshPending {
		a.refreshPending = false
		a.log("Executing pending refresh")
		a.refresh()
	}
}

func (a *app) renderScreen(buf []byte, cols int) {

	a.log("Rendering screen")

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
			a.loadData(offset)
			buildLoadingScreen(buf, cols)
			break
		}
		offset += len(a.fwd[row].data)
	}

}

func (a *app) loadData(loadFrom int) {
	const chunkSize = 128
	buf := make([]byte, chunkSize)
	var fileInfo os.FileInfo
	var n int
	go func() {

		f, err := os.Open(a.filename)
		if err != nil {
			// TODO: Handle error.
		}
		n, err = f.ReadAt(buf, int64(loadFrom))
		if err != nil && err != io.EOF {
			// TODO: Handle error.
		}
		fileInfo, err = f.Stat()
		if err != nil {
			// TODO: Handle error.
		}

		a.reactor.Enque(func() {
			a.log("Data loaded: From=%d To=%d Len=%d", loadFrom, loadFrom+n, n)
			newFileSize := int(fileInfo.Size())
			if newFileSize != a.fileSize {
				a.log("File changed: oldSize=%d newSize=%d", a.fileSize, newFileSize)
				// TODO: Invalidate everything.
				a.fileSize = newFileSize
			}
			a.fileSize = int(fileInfo.Size())

			offset := loadFrom
			for _, data := range extractLines(loadFrom, buf[:n]) {
				if len(a.fwd) == 0 && offset == a.offset {
					a.fwd = append(a.fwd, line{offset, data})
				} else if a.fwd[len(a.fwd)-1].offset+len(a.fwd[len(a.fwd)-1].data) == offset {
					a.fwd = append(a.fwd, line{offset, data})
				}
				offset += len(data)
			}

			a.refresh()
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
