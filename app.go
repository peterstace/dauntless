package main

import "fmt"

type App interface {
	Initialise()
	KeyPress(byte)
	TermSize(rows, cols int, err error)
}

type app struct {
	filename string

	rows, cols int
}

func NewApp(filename string) App {
	return &app{
		filename: filename,

		rows: -1,
		cols: -1,
	}
}

func (a *app) Initialise() {
}

func (a *app) KeyPress(b byte) {
	fmt.Printf("%c", b)
}

func (a *app) TermSize(rows, cols int, err error) {
	if a.rows != rows || a.cols != cols {
		a.rows = rows
		a.cols = cols
		fmt.Println(rows, cols)
	}
}
