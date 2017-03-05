package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

func getTTYState() (string, error) {
	cmd := exec.Command("stty", "-g")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func enterRawTTYMode() error {
	cmd := exec.Command("stty", "raw")
	cmd.Stdin = os.Stdin
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(err.Error() + ":" + string(out))
	}
	return nil
}

func restoreTTYState(state string) error {
	cmd := exec.Command("stty", state)
	cmd.Stdin = os.Stdin
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(err.Error() + ":" + string(out))
	}
	return nil
}
