package main

import (
	"os"
	"time"
)

func LoadFileSize(filename string) (int, error) {

	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return int(fileInfo.Size()), nil
}

func CollectFileSize(r Reactor, a App, filename string) {
	go func() {
		for {
			size, err := LoadFileSize(filename)
			if err != nil {
				r.Stop(err)
				return
			}
			r.Enque(func() { a.FileSize(size) })
			time.Sleep(time.Second)
		}
	}()
}
