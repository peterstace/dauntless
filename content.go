package main

import (
	"io"
	"os"
)

type Content interface {
	io.ReaderAt
	Size() int64
}

func NewFileContent(filename string) (*FileContent, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return &FileContent{f}, err
}

type FileContent struct {
	f *os.File
}

func (f *FileContent) Size() (int64, error) {
	fi, err := f.f.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}
