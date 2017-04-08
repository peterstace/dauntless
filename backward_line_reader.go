package main

import "io"

const lineReaderReadSize = 8 // TODO: This is stupidly small. Mainly for ease of testing.

func NewBackwardLineReader(reader io.ReaderAt, offset int) *BackwardLineReader {
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
