package main

import (
	"flag"
	"fmt"
	"os"
)

const version = "Dauntless <unversioned>"

func main() {

	var logfile string
	flag.StringVar(&logfile, "l", "", "debug logfile")
	vFlag := flag.Bool("v", false, "version")
	flag.Parse()

	if *vFlag {
		fmt.Println(version)
		return
	}

	if len(flag.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s <filename>\n", os.Args[0])
		os.Exit(1)
	}

	var logger Logger
	if logfile == "" {
		logger = NullLogger{}
	} else {
		var err error
		logger, err = FileLogger(logfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not open debug logfile %q: %s\n", logfile, err)
			os.Exit(1)
		}
	}

	enterAlt()
	ttyState := enterRaw()

	reactor := NewReactor(logger)

	filename := flag.Args()[0]

	screen := NewTermScreen(os.Stdout, reactor, logger)

	app := NewApp(reactor, filename, logger, screen)

	reactor.Enque(app.Initialise)
	CollectFileSize(reactor, app, filename)
	collectSignals(reactor, app)
	collectInput(reactor, app)
	collectTermSize(reactor, app)
	err := reactor.Run()

	ttyState.leaveRaw()
	leaveAlt()

	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
