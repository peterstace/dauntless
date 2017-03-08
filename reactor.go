package main

type Reactor interface {
	Enque(func())
	Run()
	Stop()
}

func NewReactor() Reactor {
	return &reactor{
		make(chan func(), 1024),
		make(chan struct{}, 1),
	}
}

// TODO: Should not have a fixed queue size.

type reactor struct {
	queue chan func()
	stop  chan struct{}
}

func (r *reactor) Enque(fn func()) {
	r.queue <- fn
}

func (r *reactor) Run() {
	for {
		// Check stopping condition first, since it has the highest priority.
		select {
		case <-r.stop:
			return
		default:
		}

		// Wait for the stopping condition, or the next event to process.
		select {
		case fn := <-r.queue:
			fn()
		case <-r.stop:
			return
		}
	}
}

func (r *reactor) Stop() {
	select {
	case r.stop <- struct{}{}:
	default:
	}
}
