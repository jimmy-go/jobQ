package jobq

import "errors"

var (
	errInvalidWorkerSize = errors.New("invalid worker size")
	errInvalidQueueSize  = errors.New("invalid queue size")
)

// Job func. Can be any function with no input vars that returns error
type Job func() error

// Dispatcher share jobs between available workers.
type Dispatcher struct {
	closed bool
	ws     chan *Worker
	queue  chan Job
	size   int
	done   chan struct{}
	wait   chan struct{}
}

// New returns a new dispatcher.
// size: how many workers would init.
// queueLen: how many jobs would put in queue.
func New(size int, queueLen int) (*Dispatcher, error) {
	if size < 1 {
		return nil, errInvalidWorkerSize
	}
	if queueLen < 1 {
		return nil, errInvalidQueueSize
	}
	d := &Dispatcher{
		ws:    make(chan *Worker, size),
		queue: make(chan Job, queueLen),
		size:  size,
		done:  make(chan struct{}, 1),
		wait:  make(chan struct{}, 1),
	}
	// init and run workers.
	for i := 0; i < d.size; i++ {
		w := newWorker(i, d.ws)
		go w.run()
		d.ws <- w
	}
	go d.run()
	return d, nil
}

// TODO; change to waitgroup

// run keep dispatching jobs between workers.
func (d *Dispatcher) run() {
	for {
		select {
		case job := <-d.queue:
			wc := <-d.ws
			select {
			case wc.jobc <- job:
				// validate queue is empty and dispatcher was stopped.
				if len(d.queue) < 1 && len(d.done) > 0 {
					for i := 0; i < d.size; i++ {
						select {
						case w := <-d.ws:
							w.stop()
						}
					}
					close(d.wait)
					return
				}
			}
		}
	}
}

// Add add job to queue channel.
func (d *Dispatcher) Add(j Job) error {
	if len(d.done) > 0 {
		return errors.New("queue closed")
	}
	d.queue <- j
	return nil
}

// Stop stops all workers.
func (d *Dispatcher) Stop() {
	d.done <- struct{}{}
}

// Wait waits until jobs are done.
func (d *Dispatcher) Wait() {
	for _ = range d.wait {
	}
}

// Worker struct implements own job channel and notifies owner dispatcher when
// is available for work.
type Worker struct {
	ID   int
	dc   chan *Worker
	jobc chan Job
	done chan struct{}
}

// newWorker returns a new worker.
func newWorker(id int, dc chan *Worker) *Worker {
	w := &Worker{
		ID:   id,
		dc:   dc,
		jobc: make(chan Job),
		done: make(chan struct{}, 1),
	}
	return w
}

func (w *Worker) stop() {
	w.done <- struct{}{}
}

// run method runs until Dispatcher.Stop() is called.
// keeps running jobs.
func (w *Worker) run() {
	for {
		select {
		case job := <-w.jobc:
			job()
			select {
			case w.dc <- w:
			}
		case <-w.done:
			return
		}
	}
}
