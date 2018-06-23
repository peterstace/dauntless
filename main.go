package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

const version = "Dauntless <unversioned>"

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

	reactor := NewReactor()
	var filename string
	var content Content

	switch len(flag.Args()) {
	case 0:
		if terminal.IsTerminal(syscall.Stdin) {
			fmt.Fprintf(os.Stderr, "Missing filename (use \"dauntless --help\" for usage)\n")
			os.Exit(1)
		}
		filename = "stdin"
		buffContent := NewBufferContent()
		CollectContent(os.Stdin, reactor, buffContent)
		content = buffContent
	case 1:
		filename = flag.Args()[0]
		var err error
		content, err = NewFileContent(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not open file %s: %s", filename, err)
			os.Exit(1)
		}
	default:
		flag.Usage()
		os.Exit(1)
	}

	mask, err := regexp.Compile(*bisectMask)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not compile regex: %v\n", err)
		os.Exit(1)
	}

	config := Config{*wrapPrefix, mask}

	enterAlt()
	ttyState := enterRaw()
	screen := NewTermScreen(os.Stdout, reactor)
	app := NewApp(reactor, content, filename, screen, config)
	reactor.Enque(app.Initialise)
	CollectFileSize(reactor, app, content)
	collectInterrupt(reactor, app)
	collectInput(reactor, app)
	CollectTermSize(reactor, app)
	err = reactor.Run()

	ttyState.leaveRaw()
	leaveAlt()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
