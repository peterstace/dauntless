package main

type Reactor interface {
	Enque(func(), string)
	Run() error
	Stop(error)
	SetPostHook(func())
	GetCycle() int
}

func NewReactor() Reactor {
	return &reactor{
		make(chan event, 1024),
		make(chan error, 1),
		0,
		nil,
	}
}

type event struct {
	action func()
	source string
}

type reactor struct {
	queue    chan event
	stop     chan error
	cycle    int
	postHook func()
}

func (r *reactor) Enque(fn func(), src string) {
	r.queue <- event{
		action: fn,
		source: src,
	}
}

func (r *reactor) Run() error {
	for {
		r.cycle++
		log.SetCycle(r.cycle)

		// Check stopping condition first, since it has the highest priority.
		select {
		case err := <-r.stop:
			return err
		default:
		}

		// Wait for the stopping condition, or the next event to process.
		select {
		case event := <-r.queue:
			log.Info("Running event from: %s", event.source)
			event.action()
			if r.postHook != nil {
				r.postHook()
			}
			err := log.Flush()
			if err != nil {
				r.Stop(err)
			}
		case err := <-r.stop:
			log.Flush()
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

func (r *reactor) SetPostHook(fn func()) {
	r.postHook = fn
}

func (r *reactor) GetCycle() int {
	return r.cycle
}
