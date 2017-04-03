package main

type App interface {
	Initialise()
	KeyPress(byte)
	TermSize(rows, cols int, err error)
	LoadComplete(LoadResponse)
}

type line struct {
	offset int
	data   string
}

type app struct {
	reactor Reactor
	log     Logger

	filename string

	loader Loader

	rows, cols int

	// Invariants:
	//  1) If fwd is populated, then offset will match the first line.
	//  2) Fwd and bck contain consecutive lines.
	offset int
	fwd    []line
	bck    []line

	fileSize int

	screenBuffer []byte
	screen       Screen
}

func NewApp(reactor Reactor, filename string, loader Loader, logger Logger, screen Screen) App {
	return &app{
		reactor:  reactor,
		filename: filename,
		loader:   loader,
		log:      logger,
		rows:     -1,
		cols:     -1,
		screen:   screen,
	}
}

func (a *app) Initialise() {
	a.log.Info("***************** Initialising log viewer ******************")
}

func (a *app) KeyPress(b byte) {

	a.log.Info("Key press: %c", b)

	switch b {
	case 'q':
		a.log.Info("Quitting.")
		a.reactor.Stop()
	case 'j':
		a.moveDown()
	case 'k':
		a.moveUp()
	case 'r':
		a.log.Info("Repainting screen.")
		a.refresh()
	case 'R':
		a.log.Info("Discarding buffered input and repainting screen.")
		a.fwd = nil
		a.bck = nil
		a.refresh()
	case 'g':
		a.moveTop()
	case 'G':
		a.moveBottom()
	default:
		a.log.Info("Unhandled key press: %d", b)
	}
}

func (a *app) moveDown() {

	a.log.Info("Moving down.")

	if len(a.fwd) == 0 {
		a.log.Warn("Cannot move down: current line not loaded.")
		return
	}

	ln := a.fwd[0]
	newOffset := ln.offset + len(ln.data)

	if newOffset == a.fileSize {
		a.log.Info("Cannot move down: reached EOF.")
		return
	}

	a.moveToOffset(newOffset)
}

func (a *app) moveUp() {

	a.log.Info("Moving up.")

	if a.offset == 0 {
		a.log.Info("Cannot move back: at start of file.")
		return
	}

	if len(a.bck) == 0 {
		a.log.Warn("Cannot move back: previous line not loaded.")
		return
	}

	a.moveToOffset(a.bck[0].offset)
}

func (a *app) moveTop() {
	a.log.Info("Jumping to start of file.")
	a.moveToOffset(0)
}

func (a *app) moveBottom() {

	a.log.Info("Jumping to bottom of file.")

	go func() {
		offset, err := FindJumpToBottomOffset(a.filename)
		if err != nil {
			a.log.Warn("Could not find jump-to-bottom offset: %v", err)
			a.reactor.Stop()
			return
		}
		a.reactor.Enque(func() { a.moveToOffset(offset) })
	}()
}

func (a *app) moveToOffset(offset int) {
	a.log.Info("Moving to offset: currentOffset=%d newOffset=%d", a.offset, offset)

	assert(offset >= 0)
	assert(offset < a.fileSize)

	if a.offset == offset {
		a.log.Info("Already at target offset.")
	} else if offset < a.offset {
		a.moveUpToOffset(offset)
	} else {
		a.moveDownToOffset(offset)
	}
}

func (a *app) moveUpToOffset(offset int) {

	a.log.Info("Moving up to offset: currentOffset=%d newOffset=%d", a.offset, offset)

	haveTargetLoaded := false
	for _, ln := range a.bck {
		if ln.offset == offset {
			haveTargetLoaded = true
			break
		}
	}
	if haveTargetLoaded {
		for a.offset != offset {
			ln := a.bck[0]
			a.fwd = append([]line{ln}, a.fwd...)
			a.bck = a.bck[1:]
			a.offset = ln.offset
		}
	} else {
		a.fwd = nil
		a.bck = nil
		a.offset = offset
	}
	a.refresh()
}

func (a *app) moveDownToOffset(offset int) {

	a.log.Info("Moving down to offset: currentOffset=%d newOffset=%d", a.offset, offset)

	haveTargetLoaded := false
	for _, ln := range a.fwd {
		if ln.offset == offset {
			haveTargetLoaded = true
			break
		}
	}
	if haveTargetLoaded {
		for a.offset != offset {
			ln := a.fwd[0]
			a.fwd = a.fwd[1:]
			a.bck = append([]line{ln}, a.bck...)
			a.offset = ln.offset + len(ln.data)
		}
	} else {
		a.fwd = nil
		a.bck = nil
		a.offset = offset
	}
	a.refresh()
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

	if a.rows < 0 || a.cols < 0 {
		a.log.Warn("Can't refresh, don't know term size yet")
		return
	}

	a.log.Info("Refreshing")

	if len(a.screenBuffer) != a.rows*a.cols {
		a.screenBuffer = make([]byte, a.rows*a.cols)
	}
	a.renderScreen()
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

func (a *app) renderScreen() {

	a.log.Info("Rendering screen.")

	for i := range a.screenBuffer {
		a.screenBuffer[i] = ' '
	}

	assert(len(a.fwd) == 0 || a.fwd[0].offset == a.offset)
	offset := a.offset
	for row := 0; row < len(a.screenBuffer)/a.cols; row++ {
		if row < len(a.fwd) {
			col := 0
			for i := 0; col+1 < a.cols && i < len(a.fwd[row].data); i++ {
				col += writeByte(a.screenBuffer[row*a.cols+col:(row+1)*a.cols], a.fwd[row].data[i], col)
			}
		} else if len(a.fwd) != 0 && a.fwd[len(a.fwd)-1].offset+len(a.fwd[len(a.fwd)-1].data) >= a.fileSize {
			// Reached end of file.
			assert(a.fwd[len(a.fwd)-1].offset+len(a.fwd[len(a.fwd)-1].data) == a.fileSize) // Assert that it's actually equal.
			break
		} else {
			a.loader.Load(offset, defaultLoadAmount)
			buildLoadingScreen(a.screenBuffer, a.cols)
			break
		}
		offset += len(a.fwd[row].data)
	}

	a.screen.Write(a.screenBuffer, a.cols)
}

const defaultLoadAmount = 64

func (a *app) LoadComplete(resp LoadResponse) {

	a.log.Info("Data loaded: From=%d To=%d Len=%d", resp.Offset, resp.Offset+len(resp.Payload), len(resp.Payload))
	if resp.FileSize != a.fileSize {
		a.log.Info("File size changed: oldSize=%d newSize=%d", a.fileSize, resp.FileSize)
		a.fileSize = resp.FileSize
	}
	a.fileSize = resp.FileSize

	offset := resp.Offset
	containedLine := false
	for _, data := range extractLines(resp.Payload) {
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
	} else if reachedEndOfFile := resp.Offset+len(resp.Payload) == a.fileSize; !reachedEndOfFile {
		a.log.Warn("Data loaded didn't contain at least one complete line: retrying with double amount.")
		a.loader.Load(resp.Offset, 2*len(resp.Payload))
	} else {
		a.log.Warn("Data loaded didn't contain at least one complete line: reached EOF")
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
