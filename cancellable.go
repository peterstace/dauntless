package main

import "sync"

type Cancellable struct {
	mu        sync.Mutex
	cancelled bool
}

func (c *Cancellable) Cancelled() bool {
	c.mu.Lock()
	can := c.cancelled
	c.mu.Unlock()
	return can
}

func (c *Cancellable) Cancel() {
	c.mu.Lock()
	c.cancelled = true
	c.mu.Unlock()
}

func (c *Cancellable) Reset() {
	c.mu.Lock()
	c.cancelled = false
	c.mu.Unlock()
}
