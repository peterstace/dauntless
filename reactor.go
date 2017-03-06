package main

type Reactor interface {
	Enque(func())
	Run()
}

func NewReactor() Reactor {
	return &reactor{make(chan func(), 1024)}
}

// TODO: Should not have a fixed queue size.

// TODO: Should be able to stop the reactor.

type reactor struct {
	queue chan func()
}

func (r *reactor) Enque(fn func()) {
	r.queue <- fn

}

func (r *reactor) Run() {
	for {
		(<-r.queue)()
	}
}
