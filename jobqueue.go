package jobq

import "errors"

var (
	errInvalidWorkerSize = errors.New("invalid worker size")
	errInvalidQueueSize  = errors.New("invalid queue size")
)

// Job it's a type with wide application than an interface{Work(id int)}
type Job func() error

// Dispatcher share jobs between workers available.
type Dispatcher struct {
	ws    chan *Worker
	queue chan Job
	size  int
	done  chan struct{}
}

// New returns a new dispatcher.
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
	}
	d.run()
	return d, nil
}

// run keep dispatching jobs between workers.
func (d *Dispatcher) run() {
	// init and run workers.
	for i := 0; i < d.size; i++ {
		w := newWorker(i, d.ws, d.done)
		go w.run()
		d.ws <- w
	}
	go func() {
		for {
			select {
			case job := <-d.queue:
				select {
				case wc := <-d.ws:
					select {
					case wc.jobc <- job:
					}
				}
			case <-d.done:
				return
			}
		}
	}()
}

// Add add job to queue channel.
func (d *Dispatcher) Add(j Job) {
	d.queue <- j
}

// Stop stops all workers.
func (d *Dispatcher) Stop() {
	// +2: we need to stop Dispatcher too.
	for i := 0; i < d.size+2; i++ {
		d.done <- struct{}{}
	}
}

// Worker struct implements own job channel and notifies owner dispatcher when is
// available for work.
type Worker struct {
	ID   int
	dc   chan *Worker
	jobc chan Job
	done chan struct{}
}

// newWorker returns a new worker.
func newWorker(id int, dc chan *Worker, donec chan struct{}) *Worker {
	w := &Worker{
		ID:   id,
		dc:   dc,
		jobc: make(chan Job),
		done: donec,
	}
	return w
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
