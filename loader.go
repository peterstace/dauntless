package main

import (
	"io"
	"os"
)

type Loader interface {
	Load(LoadRequest)
	SetHandler(LoadHandler)
}

type LoadRequest struct {
	Offset   int
	Amount   int
	Forwards bool
}

type LoadResponse struct {
	FileSize int
	Payload  []byte
	Request  LoadRequest
}

type LoadHandler interface {
	LoadComplete(LoadResponse)
}

type fileLoader struct {
	filename string
	handler  LoadHandler
	reactor  Reactor
	log      Logger
	loading  bool
}

func NewFileLoader(filename string, reactor Reactor, log Logger) Loader {
	return &fileLoader{filename, nil, reactor, log, false}
}

func (l *fileLoader) SetHandler(h LoadHandler) {
	l.handler = h
}

func (l *fileLoader) Load(req LoadRequest) {

	// Only a single loading operation allowed at a time.
	if l.loading {
		l.log.Debug("Loading already in progress.")
		return
	}
	l.loading = true

	go func() {

		f, err := os.Open(l.filename)
		if err != nil {
			l.reactor.Enque(func() {
				l.log.Warn("Could not open file: filename=%q reason=%q", l.filename, err)
				l.reactor.Stop(err)
			})
			return
		}
		defer f.Close()

		buf := make([]byte, req.Amount)

		n, err := f.ReadAt(buf, int64(req.Offset))
		if err != nil && err != io.EOF {
			l.reactor.Enque(func() {
				l.log.Warn("Could not read file: filename=%q offset=%d reason=%q", l.filename, req.Offset, err)
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
			l.loading = false
			l.handler.LoadComplete(LoadResponse{
				FileSize: int(fileInfo.Size()),
				Payload:  buf[:n],
				Request:  req,
			})
		})
	}()
}
