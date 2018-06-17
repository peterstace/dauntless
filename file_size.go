package main

import (
	"time"
)

func CollectFileSize(r Reactor, a App, c Content) {
	go func() {
		for {
			size, err := c.Size()
			if err != nil {
				r.Stop(err)
				return
			}
			r.Enque(func() { a.FileSize(int(size)) })
			time.Sleep(time.Second)
		}
	}()
}
