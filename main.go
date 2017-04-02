package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {

	var logfile string
	flag.StringVar(&logfile, "l", "", "debug logfile")
	flag.Parse()

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

	defer enterRaw().leaveRaw()

	reactor := NewReactor(logger)

	filename := flag.Args()[0]
	loader := NewFileLoader(filename, reactor, logger)

	app := NewApp(reactor, loader, logger)
	loader.SetHandler(app)

	reactor.Enque(app.Initialise)
	collectInput(reactor, app)
	collectTermSize(reactor, app)
	reactor.Run()
}
