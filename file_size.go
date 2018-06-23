package dauntless

import (
	"time"
)

func CollectFileSize(r Reactor, a App, c Content) {
	go func() {
		var lastSize int64
		var sleepFor time.Duration
		for {
			size, err := c.Size()
			if err != nil {
				r.Stop(err)
				return
			}
			resized := size != lastSize
			lastSize = size

			r.Enque(func() { a.FileSize(int(size)) })

			if resized {
				sleepFor = 0
			} else {
				sleepFor = 2 * (sleepFor + time.Millisecond)
				const maxSleep = time.Second
				if sleepFor > maxSleep {
					sleepFor = maxSleep
				}
			}
			time.Sleep(sleepFor)
		}
	}()
}
