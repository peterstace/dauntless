package main

type Reactor interface {
	Enque(func())
	Run() error
	Stop(error)
}

func NewReactor(log Logger) Reactor {
	return &reactor{
		make(chan func(), 1024),
		make(chan error, 1),
		log,
		0,
	}
}

// TODO: Should not have a fixed queue size.

type reactor struct {
	queue chan func()
	stop  chan error
	log   Logger
	cycle int
}

func (r *reactor) Enque(fn func()) {
	r.queue <- fn
}

func (r *reactor) Run() error {
	for {
		r.cycle++
		r.log.SetCycle(r.cycle)

		// Check stopping condition first, since it has the highest priority.
		select {
		case err := <-r.stop:
			return err
		default:
		}

		// Wait for the stopping condition, or the next event to process.
		select {
		case fn := <-r.queue:
			fn()
			err := r.log.Flush() // TODO: Flush in own goroutine.
			if err != nil {
				r.Stop(err)
			}
		case err := <-r.stop:
			r.log.Flush()
			return err
		}
	}
}

func (r *reactor) Stop(err error) {
	select {
	case r.stop <- err:
	default:
	}
}
