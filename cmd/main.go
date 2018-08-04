package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/peterstace/dauntless"
	"github.com/peterstace/dauntless/screen"
	"github.com/peterstace/dauntless/term"
	"golang.org/x/crypto/ssh/terminal"
)

const version = "Dauntless <unversioned>"

var log dauntless.Logger

func main() {
	var logfile string
	flag.StringVar(&logfile, "debug-logfile", "", "debug logfile")
	vFlag := flag.Bool("version", false, "version")
	wrapPrefix := flag.String("wrap-prefix", "", "prefix string for wrapped lines")
	bisectMask := flag.String("bisect-mask", "", "only consider lines matching this regex when bisecting")
	helpFlag := flag.Bool("help", false, "display help")
	flag.Parse()

	if *vFlag {
		fmt.Println(version)
		return
	}

	if *helpFlag {
		flag.Usage()
		fmt.Println()
		fmt.Println("CONTROLS:")
		fmt.Println()
		for _, ctrl := range dauntless.Controls {
			fmt.Printf("    ")
			for i, k := range ctrl.Keys {
				if i != 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%v", k)
			}
			fmt.Printf(" - %s\n\n", ctrl.Desc)
		}
		return
	}

	if logfile == "" {
		log = dauntless.NullLogger{}
	} else {
		var err error
		log, err = dauntless.FileLogger(logfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not open debug logfile %q: %s\n", logfile, err)
			os.Exit(1)
		}
	}
	dauntless.SetLogger(log)

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
		dauntless.CollectContent(os.Stdin, stop, buffContent) // TODO: Should this be here?
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

	siginterrupt := make(chan os.Signal, 1)
	signal.Notify(siginterrupt, os.Interrupt)

	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)

	tty, err := os.Open("/dev/tty")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open /dev/tty: %v", err)
		os.Exit(1)
	}

	term.EnterAlt()
	ttyState = term.EnterRaw()
	screen := screen.NewTermScreen(os.Stdout)
	app := dauntless.NewApp(content, filename, screen, config, stop, siginterrupt, tty, term.GetSize, sigwinch)
	err = app.Run()
	stop(err)
}

var ttyState term.TTYState

func stop(err error) {
	ttyState.LeaveRaw()
	term.LeaveAlt()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
