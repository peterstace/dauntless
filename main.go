package main

import (
	"fmt"
	"os"
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

	reactor := NewReactor()
	app := &app{}

	collectInput(reactor, app)

	reactor.Run()
}
