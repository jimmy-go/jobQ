// Package jobq contains type JobQ for worker pools.
package jobq

import (
	"context"
	"errors"
	"log"
)

// Worker interface defines behaviour for a worker.
type Worker interface {
	// Do runs a task and makes Worker return to pool.
	Do(ctx context.Context, v interface{}) error
}

// TaskFunc defines behavior for a task. Param v is the payload required.
type TaskFunc func() (ctx context.Context, v interface{})

// Pool contains a channel of Worker implementations.
type Pool struct {
	workersc chan Worker
	tasksc   chan TaskFunc
}

// New returns a new Pool with workers ws. Param size determines task buffer size
// before blocking.
func New(workers []Worker, size int) (*Pool, error) {
	if len(workers) == 0 {
		return nil, errors.New("empty workers")
	}
	p := &Pool{
		workersc: make(chan Worker, len(workers)),
		tasksc:   make(chan TaskFunc, size),
	}
	for i := range workers {
		x := workers[i]
		p.workersc <- x
	}
	go p.run()
	return p, nil
}

// run keep dispatching tasks between workers.
func (p *Pool) run() {
	for {
		select {
		case w := <-p.workersc:
			task := <-p.tasksc
			ctx, v := task()
			err := w.Do(ctx, v)
			if err != nil {
				log.Printf("run : err [%s]", err)
			}
			p.workersc <- w
		default:
		}
	}
}

// AddWorker adds a worker.
func (p *Pool) AddWorker(w Worker) error {
	// TODO; resize pool?.
	select {
	case p.workersc <- w:
		return nil
	default:
		return errors.New("can't add worker")
	}
}

// AddTask add a task to queue.
func (p *Pool) AddTask(task TaskFunc) error {
	select {
	case p.tasksc <- task:
		return nil
	default:
		return errors.New("queue size exceeded")
	}
}
