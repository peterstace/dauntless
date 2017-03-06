package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type ttyState string

func enterRaw() ttyState {

	cmd := exec.Command("stty", "-g")
	cmd.Stdin = os.Stdin
	oldState, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get TTY state")
		os.Exit(1)
	}

	cmd = exec.Command("stty", "cbreak", "-echo")
	cmd.Stdin = os.Stdin
	combinedOut, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not enter raw mode: %s", string(combinedOut))
		os.Exit(1)
	}

	return ttyState(strings.TrimSpace(string(oldState)))
}

func (s ttyState) leaveRaw() {

	cmd := exec.Command("stty", string(s))
	cmd.Stdin = os.Stdin
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not restore terminal state: %s", string(out))
		os.Exit(1)
	}
}
