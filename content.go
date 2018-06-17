package main

import (
	"io"
	"os"
)

type Content interface {
	Size() (int64, error)
	io.ReaderAt
}

func NewFileContent(filename string) (Content, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
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
