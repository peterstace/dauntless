package main

import (
	"bytes"
	"io"
	"os"
	"sync"
	"time"
)

type Content interface {
	Size() (int64, error)
	io.ReaderAt
}

func NewFileContent(filename string) (FileContent, error) {
	f, err := os.Open(filename)
	if err != nil {
		return FileContent{}, err
	}
	return FileContent{f}, nil
}

type FileContent struct {
	*os.File
}

func (f FileContent) Size() (int64, error) {
	fi, err := f.File.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

func NewBufferContent() *BufferContent {
	return &BufferContent{}
}

type BufferContent struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *BufferContent) Size() (int64, error) {
	s.mu.Lock()
	sz := int64(len(s.buf.Bytes()))
	s.mu.Unlock()
	return sz, nil
}

func (s *BufferContent) ReadAt(p []byte, off int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	buf := s.buf.Bytes()
	if off > int64(len(buf)) {
		return 0, io.EOF
	}
	n := copy(p, buf[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (s *BufferContent) CollectFrom(r io.Reader, reac Reactor) {
	go func() {
		buf := make([]byte, 16<<10)
		var sleepFor time.Duration
		for {
			n, err := r.Read(buf)
			if err != nil && err != io.EOF {
				reac.Stop(err)
				return
			}

			s.mu.Lock()
			s.buf.Write(buf[:n])
			s.mu.Unlock()

			if n == 0 {
				sleepFor = 2 * (sleepFor + time.Millisecond)
				const maxSleep = 500 * time.Millisecond
				if sleepFor > maxSleep {
					sleepFor = maxSleep
				}
			} else {
				sleepFor = 0
			}
			time.Sleep(sleepFor)
		}
	}()
}
