package dauntless

import (
	"bytes"
	"io"
	"os"
	"sync"
)

type Content interface {
	Size() (int64, error)
	io.ReaderAt
	Write([]byte)
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

func (f FileContent) Write([]byte) {
	panic("should not be called")
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

func (s *BufferContent) Write(p []byte) {
	s.mu.Lock()
	s.buf.Write(p)
	s.mu.Unlock()
}
