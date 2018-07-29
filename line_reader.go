package dauntless

import (
	"io"

	"github.com/peterstace/dauntless/assert"
)

const lineReaderReadSize = 1 << 12

type LineReader interface {
	ReadLine() (string, error)
}

func NewForwardLineReader(reader io.ReaderAt, offset int) *ForwardLineReader {
	return &ForwardLineReader{reader, offset, make([]byte, lineReaderReadSize), nil}
}

type ForwardLineReader struct {
	reader  io.ReaderAt
	offset  int
	readBuf []byte
	unused  []byte
}

func (f *ForwardLineReader) ReadLine() (string, error) {
	// Check if the new newline is in the unused buffer.
	for i, b := range f.unused {
		if b == '\n' {
			line := f.unused[:i+1]
			f.unused = f.unused[i+1:]
			return string(line), nil
		}
	}

	// Copy a new set of bytes into unused.
	n, err := f.reader.ReadAt(f.readBuf, int64(f.offset))
	if err != nil && (err != io.EOF || n == 0) {
		return "", err
	}
	f.offset += n
	f.unused = append(f.unused, f.readBuf[:n]...)
	return f.ReadLine()
}

func NewBackwardLineReader(reader io.ReaderAt, offset int) *BackwardLineReader {
	return &BackwardLineReader{reader, offset, make([]byte, lineReaderReadSize), nil}
}

type BackwardLineReader struct {
	reader  io.ReaderAt
	offset  int
	readBuf []byte
	unused  []byte
}

func (b *BackwardLineReader) ReadLine() (string, error) {
	if len(b.unused) == 0 && b.offset == 0 {
		return "", io.EOF
	}

	for i := len(b.unused) - 1; i >= 0; i-- {
		if b.offset == 0 && i == 0 {
			line := string(b.unused)
			b.unused = nil
			return line, nil
		}
		if b.unused[i] == '\n' && i != len(b.unused)-1 {
			line := string(b.unused[i+1:])
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
		return "", err
	}
	b.offset -= n
	assert.True(b.offset >= 0)
	oldUnused := b.unused
	b.unused = make([]byte, n+len(b.unused))
	copy(b.unused, b.readBuf[:n])
	copy(b.unused[n:], oldUnused)

	return b.ReadLine()
}
