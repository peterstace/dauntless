package term

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type ttyState string

var tty *os.File

func init() {
	var err error
	tty, err = os.Open("/dev/tty")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open /dev/tty: %v", err)
		os.Exit(1)
	}
}

func EnterRaw() ttyState {
	cmd := exec.Command("stty", "-g")
	cmd.Stdin = tty
	oldState, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get TTY state")
		os.Exit(1)
	}

	cmd = exec.Command("stty", "cbreak", "-echo")
	cmd.Stdin = tty
	combinedOut, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not enter raw mode: %s", string(combinedOut))
		os.Exit(1)
	}

	return ttyState(strings.TrimSpace(string(oldState)))
}

func (s ttyState) LeaveRaw() {
	cmd := exec.Command("stty", string(s))
	cmd.Stdin = tty
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not restore terminal state: %s", string(out))
		os.Exit(1)
	}
}

func EnterAlt() {
	cmd := exec.Command("tput", "smcup")
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not enter alt buffer.")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, string(out))
}

func LeaveAlt() {
	cmd := exec.Command("tput", "rmcup")
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not leave alt buffer.")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, string(out))
}

func GetTermSize() (rows int, cols int, err error) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = tty
	var dim []byte
	dim, err = cmd.Output()
	if err == nil {
		_, err = fmt.Sscanf(string(dim), "%d %d", &rows, &cols)
	}
	return
}
