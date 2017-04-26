package main

import (
	"fmt"
	"regexp"
	"strconv"
)

type CommandMode interface {
	Entered(string, App)
	Prompt() string
}

type search struct{}

func (search) Entered(cmd string, a App) {
	re, err := regexp.Compile(cmd)
	if err != nil {
		a.CommandFailed(err)
		return
	}
	a.SearchCommandEntered(re)
}

func (search) Prompt() string {
	return "Enter search regexp (interrupt to cancel): "
}

type colour struct{}

var styles = [...]Style{Default, Black, Red, Green, Yellow, Blue, Magenta, Cyan, White}

func (colour) Entered(cmd string, a App) {

	err := fmt.Errorf("colour code must be in format [0-8][0-8]: %v", cmd)
	if len(cmd) != 2 {
		a.CommandFailed(err)
		return
	}
	fg := cmd[0]
	bg := cmd[1]
	if fg < '0' || fg > '8' || bg < '0' || bg > '8' {
		a.CommandFailed(err)
		return
	}

	a.ColourCommandEntered(MixStyle(styles[fg-'0'], styles[bg-'0']))
}

func (colour) Prompt() string {
	return "Enter colour code (interrupt to cancel): "
}

type seek struct{}

func (seek) Entered(cmd string, a App) {
	seekPct, err := strconv.ParseFloat(cmd, 64)
	if err != nil {
		a.CommandFailed(err)
		return
	}
	if seekPct < 0 || seekPct > 100 {
		a.CommandFailed(fmt.Errorf("seek percentage out of range [0, 100]: %v", seekPct))
		return
	}
	a.SeekCommandEntered(seekPct)
}

func (seek) Prompt() string {
	return "Enter seek percentage (interrupt to cancel): "
}

type bisect struct{}

func (bisect) Entered(cmd string, a App) {
	a.BisectCommandEntered(cmd)
}

func (bisect) Prompt() string {
	return "Enter bisect target (interrupt to cancel): "
}

type quit struct{}

func (quit) Entered(cmd string, a App) {
	switch cmd {
	case "y":
		a.QuitCommandEntered(true)
	case "n":
		a.QuitCommandEntered(false)
	default:
		a.CommandFailed(fmt.Errorf("invalid quit response (should be y/n): %v", cmd))
	}
}

func (quit) Prompt() string {
	return "Do you really want to quit? (y/n): "
}
