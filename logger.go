package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"
)

type Logger interface {
	Info(format string, args ...interface{})
	Debug(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Flush() error
}

type NullLogger struct{}

func (NullLogger) Info(format string, args ...interface{})  {}
func (NullLogger) Debug(format string, args ...interface{}) {}
func (NullLogger) Warn(format string, args ...interface{})  {}
func (NullLogger) Flush() error                             { return nil }

func FileLogger(filepath string) (Logger, error) {
	f, err := os.OpenFile(filepath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0664)
	if err != nil {
		return nil, err
	}
	return &fileLogger{
		buf:  new(bytes.Buffer),
		file: f,
	}, nil
}

type fileLogger struct {
	buf  *bytes.Buffer
	file *os.File
	err  error
}

func (f *fileLogger) Info(format string, args ...interface{}) {
	f.log("[Info ] "+format, args...)
	f.Flush()
}

func (f *fileLogger) Debug(format string, args ...interface{}) {
	f.log("[Debug] "+format, args...)
	f.Flush()
}

func (f *fileLogger) Warn(format string, args ...interface{}) {
	f.log("[Warn ] "+format, args...)
	f.Flush()
}

func (f *fileLogger) log(format string, args ...interface{}) {
	format = fmt.Sprintf("%s %s\n", time.Now().Format("15:04:05.000000"), format)
	_, f.err = fmt.Fprintf(f.buf, format, args...)
}

func (f *fileLogger) Flush() error {
	if f.err != nil {
		return f.err
	}
	_, err := io.Copy(f.file, f.buf)
	f.buf.Reset()
	return err
}
