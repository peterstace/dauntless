package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"syscall"

	"github.com/peterstace/dauntless"
	"github.com/peterstace/dauntless/term"
	"golang.org/x/crypto/ssh/terminal"
)

const version = "Dauntless <unversioned>"

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

	if logfile != "" {
		lg, err := dauntless.FileLogger(logfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not open debug logfile %q: %s\n", logfile, err)
			os.Exit(1)
		}
		dauntless.SetLogger(lg)
	}

	reactor := dauntless.NewReactor()
	var filename string
	var content dauntless.Content

	switch len(flag.Args()) {
	case 0:
		if terminal.IsTerminal(syscall.Stdin) {
			fmt.Fprintf(os.Stderr, "Missing filename (use \"dauntless --help\" for usage)\n")
			os.Exit(1)
		}
		filename = "stdin"
		buffContent := dauntless.NewBufferContent()
		buffContent.CollectFrom(os.Stdin, reactor)
		content = buffContent
	case 1:
		filename = flag.Args()[0]
		var err error
		content, err = dauntless.NewFileContent(filename)
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

	config := dauntless.Config{*wrapPrefix, mask}

	term.EnterAlt()
	ttyState := term.EnterRaw()
	screen := dauntless.NewTermScreen(os.Stdout, reactor)
	app := dauntless.NewApp(reactor, content, filename, screen, config)
	reactor.Enque(app.Initialise)
	dauntless.CollectFileSize(reactor, app, content)
	dauntless.CollectInterrupt(reactor, app)
	dauntless.CollectInput(reactor, app)
	dauntless.CollectTermSize(reactor, app)
	err = reactor.Run()

	ttyState.LeaveRaw()
	term.LeaveAlt()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
