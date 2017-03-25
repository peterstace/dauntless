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

	fileSize int

	refreshInProgress bool
	refreshPending    bool
	screenBuffer      []byte

	skipList *skipList
}

func NewApp(reactor Reactor, filename string, logger Logger) App {
	return &app{
		reactor:  reactor,
		filename: filename,
		log:      logger,

		rows: -1,
		cols: -1,

		fileSize: 0,

		skipList: newSkipList(1), // TODO: Should be higher for performance.
	}
}

func (a *app) Initialise() {
	a.log("***************** Initialising log viewer ******************")
}

func (a *app) KeyPress(b byte) {
	a.log("Key press: %d", b)

	switch b {
	case 'j':
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

	a.renderScreen(a.screenBuffer, a.cols)

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
}

func byteRepr(b byte) string {
	switch b {
	case '\n':
		return ""
	}
	if b < 32 || b >= 127 {
		return "."
	}
	return string([]byte{b})
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

	var row int

	var missingData bool
	var previousElement *element
	var currentElement *element
	for {

		// Get the next (or first) element.
		if currentElement == nil {
			currentElement = a.skipList.find(a.positionOffset)
			a.log("First element: %p", currentElement)
		} else {
			previousElement = currentElement
			currentElement = currentElement.next[0]
			a.log("Next element: %p", currentElement)
		}

		// Make sure we actually got an element.
		if currentElement == nil {
			if previousElement == nil {
				a.log("Missing data: no previous element")
				missingData = true
				break
			} else if previousElement.offset+len(previousElement.data) < a.fileSize {
				a.log("Missing data: didn't reach EOF")
				missingData = true
				break
			} else {
				a.log("Missing data: but at end of file")
				break
			}

		}

		// Make sure the element follows from the previous element.
		if previousElement != nil && previousElement.offset+len(previousElement.data) != currentElement.offset {
			missingData = true
			break
		}

		// Render the line.
		col := 0
		for i := 0; i < cols && i < len(currentElement.data); i++ {
			rep := byteRepr(currentElement.data[i])
			copy(buf[row*cols+col:(row+1)*cols], rep)
			col += len(rep)
		}
		row++
	}

	if missingData {
		a.log("Missing data, rendering loading screen")
		buildLoadingScreen(buf, cols)
		var loadFrom int
		if previousElement != nil {
			loadFrom = previousElement.offset + len(previousElement.data)
		}
		a.loadData(loadFrom)
	}
}

func (a *app) loadData(loadFrom int) {
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
			for _, line := range extractLines(loadFrom, buf[:n]) {
				if a.skipList.find(offset) == nil {
					a.log("Inserting line into skip list: offset=%d data=%q", offset, line)
					a.skipList.insert(offset, line)
				} else {
					a.log("Line already in skip list: offset=%d data=%q", offset, line)
				}
				offset += len(line)
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
