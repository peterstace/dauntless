package main

import (
	"fmt"
	"os"
	"time"
)

func main() {

	ttyState, err := getTTYState()
	if err != nil {
		fmt.Printf("Could not get TTY state: %s\n", err)
		os.Exit(1)
	}

	if err := enterRawTTYMode(); err != nil {
		fmt.Printf("Could not enter raw TTY mode: %s\n", err)
		os.Exit(1)
	}

	defer func() {
		if err := restoreTTYState(ttyState); err != nil {
			fmt.Printf("Could not restore TTY mode: %s\n", err)
			os.Exit(1)
		}
	}()

	go func() {
		for {
			var buf [4]byte
			n, err := os.Stdin.Read(buf[:])
			if err != nil {
				// TODO:
				panic(err)
			}
			fmt.Println(buf[:n])
		}
	}()

	time.Sleep(10 * time.Second)
}
