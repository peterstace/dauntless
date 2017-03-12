package main

type App interface {
	Initialise()
	KeyPress(byte)
	TermSize(rows, cols int, err error)
}

type app struct {
	reactor  Reactor
	filename string
	log      Logger

	rows, cols int

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

	// Creating buffer.
	for i := range a.screenBuffer {
		a.screenBuffer[i] = ' '
	}
	for i := 0; i < 26; i++ {
		a.screenBuffer[a.cols+3+i] = byte('A' + i)
	}
	for i := 0; i < 10; i++ {
		a.screenBuffer[a.cols*2+3+i] = byte('0' + i)
	}

	a.log("Writing to screen")
	go func() {
		WriteToTerm(a.screenBuffer)
		a.reactor.Enque(a.notifyRefreshComplete)
	}()
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
