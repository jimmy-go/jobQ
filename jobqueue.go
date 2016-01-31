package jobq

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

// NewWorker returns a new worker.
func NewWorker(id int, workerPool chan chan Job, errc chan error, dq chan Job) (*Worker, error) {
	return &Worker{
		ID:         id,
		WorkerPool: workerPool,
		JobChannel: make(chan Job),
		Errc:       errc,
		DQ:         dq,
		done:       make(chan struct{}),
	}, nil
}

// Run method starts the run loop for the worker, listening for a quit channel
// in case we need to stop it
func (w *Worker) Run() {
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

// Stop signals the worker to stop listening for work requests.
func (w *Worker) Stop() {
	select {
	case w.done <- struct{}{}:
	default:
	}
}

// Dispatcher manage job queue between workers.
type Dispatcher struct {
	WorkerPool chan chan Job
	Queue      chan Job
	Max        int
	Errc       chan error
}

// NewDispatcher returns a new dispatcher.
func NewDispatcher(maxWorkers int, queue chan Job, errc chan error) *Dispatcher {
	return &Dispatcher{
		WorkerPool: make(chan chan Job, maxWorkers),
		Max:        maxWorkers,
		Queue:      queue,
		Errc:       errc,
	}
}

// Run inits dispatcher job.
func (d *Dispatcher) Run() {
	// init and run workers.
	for i := 0; i < d.Max; i++ {
		w, err := NewWorker(i+1, d.WorkerPool, d.Errc, d.Queue)
		if err != nil {
			/// TODO; manage errors
			panic(err)
		}
		w.Run()
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
			/*
				go func(job Job) {
					// take a worker channel.
					jobChannel := <-d.WorkerPool

					// dispatch the job to the worker job channel
					jobChannel <- job
				}(job)
			*/
		}
	}
}
