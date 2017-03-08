package main

import (
	"fmt"
	"os"
)

func main() {

	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <filename>\n", os.Args[0])
		os.Exit(1)
	}
	filename := os.Args[1]

	defer enterRaw().leaveRaw()
	reactor := NewReactor()
	app := NewApp(filename)
	collectInput(reactor, app)
	app.Initialise()
	reactor.Run()
}
