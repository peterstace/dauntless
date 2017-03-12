package main

import "fmt"

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
	}
}

func (a *app) Initialise() {
	a.log("***************** Initialising log viewer ******************")
}

func (a *app) KeyPress(b byte) {
	a.log("Key press: %d", b)
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

	/* --- */
	if screenSlice, err := a.calculateScreenSlice(); err != nil {
		// TODO: Request that the unloaded chunk + the next one are loaded.
		buildLoadingScreen(a.screenBuffer, a.cols)
	} else {
		// TODO: Use the screen slice to display the screen.
		_ = screenSlice
	}
	/* --- */

	a.log("Writing to screen")
	go func() {
		WriteToTerm(a.screenBuffer)
		a.reactor.Enque(a.notifyRefreshComplete)
	}()
}

type unloadedChunkError int

func (e unloadedChunkError) Error() string {
	return fmt.Sprintf("chunk %d is unloaded", int(e))
}

// Gets all of the parts of the file needed to display the screen. If a
// required chunk isn't loaded, an error is returned.
func (a *app) calculateScreenSlice() ([]byte, error) {

	// Get the chunk that contains the current position.
	startChunkIdx := a.positionOffset / chunkSize
	chunk, ok := a.chunks[startChunkIdx]
	if !ok {
		return nil, unloadedChunkError(startChunkIdx)
	}

	// Partial chunk at end of file. So the chunk is all that's needed to
	// display the screen.
	if len(chunk) < chunkSize {
		return chunk, nil
	}

	// Full chunk. Check to see if it contains a screen's worth of data.
	assert(len(chunk) == chunkSize)
	newLineCount := 0
	enoughData := false
	for i := a.positionOffset - startChunkIdx*chunkSize; i < len(chunk); i++ {
		if chunk[i] == '\n' {
			newLineCount++
			if newLineCount == a.rows {
				enoughData = true
				break
			}
		}
	}
	if enoughData {
		return chunk, nil
	}

	// Screen spans multiple chunks. Build a new slice containing a copy of the
	// data. We have to do this because chunks may not be in contiguous memory.
	// TODO
	return nil, nil
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

func buildLoadingScreen(buf []byte, cols int) {
	for i := range buf {
		buf[i] = ' '
	}
	const loading = "Loading..."
	row := len(buf) / cols / 2
	startCol := (cols - len(loading)) / 2
	copy(buf[row*cols+startCol:], loading)
}
