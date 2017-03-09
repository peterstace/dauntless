package main

import (
	"fmt"
	"os"
	"time"
)

type Logger func(format string, args ...interface{})

var NullLogger = func(format string, args ...interface{}) {}

func FileLogger(filepath string) (Logger, error) {
	f, err := os.OpenFile(filepath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0664)
	return func(format string, args ...interface{}) {
		// Ignore errors. They're awkward to handle for logging, and logging is
		// only best effort anyway.
		format = fmt.Sprintf("%s %s\n", time.Now().Format("15:04:05.000000000"), format)
		fmt.Fprintf(f, format, args...)
		f.Sync()
	}, err
}
