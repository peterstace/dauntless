package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
)

const version = "Dauntless 0.8.1"

var log Logger

func main() {
	var logfile string
	flag.StringVar(&logfile, "debug-logfile", "", "debug logfile")
	vFlag := flag.Bool("version", false, "version")
	wrapPrefix := flag.String("wrap-prefix", "", "prefix string for wrapped lines")
	bisectMask := flag.String("bisect-mask", "", "only consider lines matching this regex when bisecting")
	flag.Parse()

	if *vFlag {
		fmt.Println(version)
		return
	}

	if len(flag.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s <filename>\n", os.Args[0])
		os.Exit(1)
	}

	mask, err := regexp.Compile(*bisectMask)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not compile regex: %v\n", err)
		os.Exit(1)
	}

	config := Config{*wrapPrefix, mask}

	if logfile == "" {
		log = NullLogger{}
	} else {
		var err error
		log, err = FileLogger(logfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not open debug logfile %q: %s\n", logfile, err)
			os.Exit(1)
		}
	}

	enterAlt()
	ttyState := enterRaw()

	reactor := NewReactor()

	filename := flag.Args()[0]

	screen := NewTermScreen(os.Stdout, reactor)

	app := NewApp(reactor, filename, screen, config)

	reactor.Enque(app.Initialise)
	CollectFileSize(reactor, app, filename)
	collectInterrupt(reactor, app)
	collectInput(reactor, app)
	collectTermSize(reactor, app)
	err = reactor.Run()

	ttyState.leaveRaw()
	leaveAlt()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
