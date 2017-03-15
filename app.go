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

const chunkSize = 128

type app struct {
	reactor  Reactor
	filename string
	log      Logger

	rows, cols int

	positionOffset int

	chunks map[int][]byte

	refreshInProgress bool
	refreshPending    bool
	screenBuffer      []byte
}

func NewApp(reactor Reactor, filename string, logger Logger) App {
	return &app{
		reactor:  reactor,
		filename: filename,
		log:      logger,

		rows: -1,
		cols: -1,

		chunks: make(map[int][]byte),
	}
}

func (a *app) Initialise() {
	a.log("***************** Initialising log viewer ******************")
}

func (a *app) KeyPress(b byte) {
	a.log("Key press: %d", b)

	switch b {
	case 'j':
		a.log("Calculating start of next line")
		n, ok := findStartOfNextLine(a.positionOffset, a.chunks)
		a.log("Result: CurrentOffset=%d StartOfNextLine=%d", a.positionOffset, n)
		a.log("TEST: %v", a.chunks[0])
		if ok {
			a.positionOffset = n
			a.refresh()
		} else {
			// bell?
		}
	case 'k':
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

	if screenSlice, err := extractLines(a.positionOffset, a.rows, a.chunks); err != nil {
		a.log("Cannot show file data: %s", err)
		chunkIdx := int(err.(unloadedChunkError))
		a.loadChunk(chunkIdx)
		buildLoadingScreen(a.screenBuffer, a.cols)
	} else {
		a.log("Building screen buffer")
		buildDataScreen(a.screenBuffer, a.cols, screenSlice)
	}

	a.log("Writing to screen")
	go func() {
		WriteToTerm(a.screenBuffer)
		a.reactor.Enque(a.notifyRefreshComplete)
	}()
}

func buildDataScreen(buf []byte, cols int, screenSlice []byte) {
	for i := range buf {
		buf[i] = ' '
	}
	offset := 0
	for row := 0; row < len(buf)/cols; row++ {
		n := mustFindNewLine(screenSlice[offset:])
		line := screenSlice[offset : offset+n]
		visiblePartOfLine := line[:min(cols, len(line))]

		for i := range visiblePartOfLine {
			// TODO: Doesn't handle zero width and tabs correctly.
			buf[row*cols+i] = byteRepr(visiblePartOfLine[i])
		}

		offset += n
		offset++ // Advance past the newline.
		if offset == len(screenSlice) {
			for r := row + 1; r < len(buf)/cols; r++ {
				buf[cols*r] = '~'
			}
			break
		}
	}
}

func byteRepr(b byte) byte {
	if b < 32 || b >= 127 {
		return '.'
	}
	return b
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

func (a *app) loadChunk(chunkIdx int) {
	buf := make([]byte, chunkSize)
	var n int
	go func() {
		f, err := os.Open(a.filename)
		if err != nil {
			// TODO: Handle error.
		}
		n, err = f.ReadAt(buf, int64(chunkIdx*chunkSize))
		if err != nil && err != io.EOF {
			// TODO: Handle error.
		}
	}()
	a.reactor.Enque(func() {
		a.log("Chunk loaded: Idx=%d", chunkIdx)
		a.chunks[chunkIdx] = buf[:n]
		a.refresh()
	})
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
