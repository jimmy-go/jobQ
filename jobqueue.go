package jobq

import "errors"

var (
	errInvalidWorkerSize = errors.New("invalid worker size")
	errInvalidQueueSize  = errors.New("invalid queue size")
)

// Job type for job.
type Job func() error

// Worker worker executing jobs work.
type Worker struct {
	ID         int
	WorkerPool chan chan Job
	JobChannel chan Job
	// Dispatcher queue channel.
	DQ   chan Job
	Errc chan error
	done chan struct{}
}

// newWorker returns a new worker.
func newWorker(id int, workerPool chan chan Job, errc chan error, dq chan Job,
	donec chan struct{}) *Worker {
	w := &Worker{
		ID:         id,
		WorkerPool: workerPool,
		JobChannel: make(chan Job),
		Errc:       errc,
		DQ:         dq,
		done:       donec,
	}
	return w
}

// run method starts the run loop for the worker, listening for a quit channel
// in case we need to stop it
func (w *Worker) run() {
	go func() {
		// w.WorkerPool <- w.JobChannel
		for {
			// register the current worker into the worker queue.
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				select {
				case w.Errc <- job():
				}
			case <-w.done:
				// TODO, return worker queue to dispatcher job queue.
				return
			}
		}
	}()
}

// Dispatcher manage job queue between workers.
type Dispatcher struct {
	WorkerPool chan chan Job
	Queue      chan Job
	Size       int
	Errc       chan error
	done       chan struct{}
}

// New returns a new dispatcher.
func New(size int, queueLen int, errc chan error) (*Dispatcher, error) {
	if size < 1 {
		return nil, errInvalidWorkerSize
	}
	if queueLen < 1 {
		return nil, errInvalidQueueSize
	}
	d := &Dispatcher{
		WorkerPool: make(chan chan Job, size),
		Queue:      make(chan Job, queueLen),
		Size:       size,
		Errc:       errc,
		done:       make(chan struct{}, 1),
	}
	d.run()
	return d, nil
}

// run inits dispatcher job.
func (d *Dispatcher) run() {
	// init and run workers.
	for i := 0; i < d.Size; i++ {
		w := newWorker(i+1, d.WorkerPool, d.Errc, d.Queue, d.done)
		w.run()
	}
	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-d.Queue:
			select {
			case jobc := <-d.WorkerPool:
				select {
				case jobc <- job:
				}
			}
		}
	}
}

// Add adds a task for the job queue.
func (d *Dispatcher) Add(j Job) {
	select {
	case d.Queue <- j:
	}
}

// Stop stops all works and return all go routines.
func (d *Dispatcher) Stop() {
	for i := 0; i < d.Size; i++ {
		select {
		case d.done <- struct{}{}:
		}
	}
}
