package main

import "fmt"

type App interface {
	Initialise()
	KeyPress(byte)
	Size(rows, cols int)
}

type app struct {
	filename string
}

func NewApp(filename string) App {
	return &app{
		filename,
	}
}

func (a *app) Initialise() {
}

func (a *app) KeyPress(b byte) {
	fmt.Printf("%c", b)
}

func (a *app) Size(rows, cols int) {
}
