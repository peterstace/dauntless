package main

type App interface {
	Initialise()
	KeyPress(byte)
	TermSize(rows, cols int, err error)
}

type app struct {
	filename   string
	log        Logger
	rows, cols int
}

func NewApp(filename string, logger Logger) App {
	return &app{
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
	a.log("Key press: %d", b)
}

func (a *app) TermSize(rows, cols int, err error) {
	if a.rows != rows || a.cols != cols {
		a.rows = rows
		a.cols = cols
		a.log("Term size: rows=%d cols=%d", rows, cols)
	}
}
