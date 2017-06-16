package main

import (
	"bufio"
	"io"
	"os"
)

type LineReader interface {
	ReadLine() ([]byte, error)
}

func NewForwardLineReader(f *os.File, offset int) *ForwardLineReader {
	return &ForwardLineReader{f, offset, nil}
}

type ForwardLineReader struct {
	file   *os.File
	start  int
	reader *bufio.Reader
}

func (f *ForwardLineReader) ReadLine() ([]byte, error) {
	if f.reader == nil {
		if _, err := f.file.Seek(int64(f.start), 0); err != nil {
			return nil, err
		}
		f.reader = bufio.NewReader(f.file)
	}
	return f.reader.ReadBytes('\n')
}

func NewBackwardLineReader(reader io.ReaderAt, offset int) *BackwardLineReader {
	const lineReaderReadSize = 1 << 12
	return &BackwardLineReader{reader, offset, make([]byte, lineReaderReadSize), nil}
}

type BackwardLineReader struct {
	reader  io.ReaderAt
	offset  int
	readBuf []byte
	unused  []byte
}

func (b *BackwardLineReader) ReadLine() ([]byte, error) {

	if len(b.unused) == 0 && b.offset == 0 {
		return nil, io.EOF
	}

	for i := len(b.unused) - 1; i >= 0; i-- {
		if b.offset == 0 && i == 0 {
			line := b.unused
			b.unused = nil
			return line, nil
		}
		if b.unused[i] == '\n' && i != len(b.unused)-1 {
			line := b.unused[i+1:]
			b.unused = b.unused[:i+1]
			return line, nil
		}
	}

	readFrom := b.offset - len(b.readBuf)
	if readFrom < 0 {
		b.readBuf = b.readBuf[:len(b.readBuf)+readFrom]
		readFrom = 0
	}
	n, err := b.reader.ReadAt(b.readBuf, int64(readFrom))
	if err != nil && err != io.EOF {
		return nil, err
	}
	b.offset -= n
	assert(b.offset >= 0)
	oldUnused := b.unused
	b.unused = make([]byte, n+len(b.unused))
	copy(b.unused, b.readBuf[:n])
	copy(b.unused[n:], oldUnused)

	return b.ReadLine()
}
