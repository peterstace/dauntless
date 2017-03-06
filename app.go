package main

import "fmt"

type Multibyte int

const (
	ArrowUp Multibyte = iota
	ArrowDown
	ArrowLeft
	ArrowRight
)

type App interface {
	KeyPress(byte)
	Size(rows, cols int)
}

type app struct{}

func (a *app) KeyPress(b byte) {
	fmt.Printf("%c", b)
}

func (a *app) Size(rows, cols int) {
}
