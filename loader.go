package main

import (
	"io"
	"os"
)

type Loader interface {
	Load(offset int, size int)
	SetHandler(LoadHandler)
}

type LoadResponse struct {
	FileSize        int
	Offset          int
	RequestedAmount int
	Payload         []byte
}

type LoadHandler interface {
	LoadComplete(LoadResponse)
}

type fileLoader struct {
	filename string
	handler  LoadHandler
	reactor  Reactor
	log      Logger
}

func NewFileLoader(filename string, reactor Reactor, log Logger) Loader {
	return &fileLoader{filename, nil, reactor, log}
}

func (l *fileLoader) SetHandler(h LoadHandler) {
	l.handler = h
}

func (l *fileLoader) Load(offset int, size int) {
	go func() {

		f, err := os.Open(l.filename)
		if err != nil {
			l.reactor.Enque(func() {
				l.log.Warn("Could not open file: filename=%q reason=%q", l.filename, err)
				l.reactor.Stop(err)
			})
		}

		buf := make([]byte, size)

		n, err := f.ReadAt(buf, int64(offset))
		if err != nil && err != io.EOF {
			l.reactor.Enque(func() {
				l.log.Warn("Could not read file: filename=%q offset=%d reason=%q", l.filename, offset, err)
				l.reactor.Stop(err)
			})
			return
		}

		fileInfo, err := f.Stat()
		if err != nil {
			l.reactor.Enque(func() {
				l.log.Warn("Could not stat file: filename=%q reason=%q", l.filename, err)
				l.reactor.Stop(err)
			})
			return
		}

		l.reactor.Enque(func() {
			l.handler.LoadComplete(LoadResponse{
				FileSize:        int(fileInfo.Size()),
				Offset:          offset,
				RequestedAmount: size,
				Payload:         buf[:n],
			})
		})
	}()
}
