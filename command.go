package main

import (
	"fmt"
	"regexp"
	"strconv"
)

type CommandMode interface {
	Entered(string, CommandHandler)
}

type search struct{}

func (search) Entered(cmd string, h CommandHandler) {
	re, err := regexp.Compile(cmd)
	if err != nil {
		h.CommandFailed(err)
		return
	}
	h.SearchCommandEntered(re)
}

type colour struct{}

var styles = [...]Style{Default, Black, Red, Green, Yellow, Blue, Magenta, Cyan, White}

func (colour) Entered(cmd string, h CommandHandler) {

	err := fmt.Errorf("colour code must be in format [0-8][0-8]: %v", cmd)
	if len(cmd) != 2 {
		h.CommandFailed(err)
		return
	}
	fg := cmd[0]
	bg := cmd[1]
	if fg < '0' || fg > '8' || bg < '0' || bg > '8' {
		h.CommandFailed(err)
		return
	}

	h.ColourCommandEntered(MixStyle(styles[fg-'0'], styles[bg-'0']))
}

type seek struct{}

func (seek) Entered(cmd string, h CommandHandler) {
	seekPct, err := strconv.ParseFloat(cmd, 64)
	if err != nil {
		h.CommandFailed(err)
		return
	}
	if seekPct < 0 || seekPct > 100 {
		h.CommandFailed(fmt.Errorf("seek percentage out of range [0, 100]: %v", seekPct))
		return
	}
	h.SeekCommandEntered(seekPct)
}

type bisect struct{}

func (bisect) Entered(cmd string, h CommandHandler) {
	h.BisectCommandEntered(cmd)
}

type quit struct{}

func (quit) Entered(cmd string, h CommandHandler) {
	switch cmd {
	case "y":
		h.QuitCommandEntered(true)
	case "n":
		h.QuitCommandEntered(false)
	default:
		h.CommandFailed(fmt.Errorf("invalid quit response (should be y/n): %v", cmd))
	}
}
