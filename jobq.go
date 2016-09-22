// Package jobq contains type JobQ for worker pools.
//
// The MIT License (MIT)
//
// Copyright (c) 2016 Angel Del Castillo
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package jobq

import (
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// TaskFunc defines behavior for a task.
type TaskFunc func(chan struct{}) error

var (
	// DefaultTimeout is default duration for tasks.
	DefaultTimeout = time.Duration(1 * time.Second)

	// ErrStopped error is returned when no more tasks are
	// allowed to run on JobQ.
	ErrStopped = errors.New("jobq: no more task allowed")

	// ErrTimeout error is returned when a task timeouts.
	ErrTimeout = errors.New("jobq: task timeout")

	// ErrInvalidWorkerSize error is returned when you init
	// a new JobQ with less than 1 worker(s).
	ErrInvalidWorkerSize = errors.New("jobq: invalid worker size")

	// ErrInvalidQueueSize error is returned when you init
	// a new JobQ with queue length less than 1.
	ErrInvalidQueueSize = errors.New("jobq: invalid queue size")

	// ErrQueueSizeExceeded error is returned to prevent
	// tasks channel block.
	ErrQueueSizeExceeded = errors.New("jobq: queue size exceeded")

	// ErrQuit error returned when Stop method is called.
	ErrQuit = errors.New("jobq: pool is quitting")

	// ErrAddWorker error returned when Populate fails.
	ErrAddWorker = errors.New("jobq: can't add worker")
)

const (
	statusExit = int32(iota + 1)
	statusStopping
)

// JobQ share jobs between available workers.
type JobQ struct {
	// workersc contain all workers from JobQ.
	workersc chan Worker

	// tasksc contain all tasks to run.
	// When n tasks is equal to tasks capacity it will return
	// error to prevent blocks.
	tasksc chan TaskFunc

	// wg is needed to know when JobQ is ending all his
	// tasks. It's unlocked by Stop method.
	wg sync.WaitGroup

	// status inner status for work execution in run method.
	status int32
}

// New returns a new empty JobQ. You need to add workers calling
// Populate method.
//
// workers: workers to be used.
// queueSize: tasks channel capacity.
func New(workers, queueSize int, timeout time.Duration) (*JobQ, error) {
	if workers < 1 {
		return nil, ErrInvalidWorkerSize
	}
	if queueSize < 1 {
		return nil, ErrInvalidQueueSize
	}
	d := &JobQ{
		tasksc:   make(chan TaskFunc, queueSize),
		workersc: make(chan Worker, 5000),
	}
	d.wg.Add(1)

	go d.run()
	return d, nil
}

// Must returns a new JobQ or panics.
func Must(size, queueSize int, timeout time.Duration) *JobQ {
	d, err := New(size, queueSize, timeout)
	if err != nil {
		panic(err)
	}
	return d
}

// run keep dispatching tasks between workers.
func (d *JobQ) run() {

	for {
		state := atomic.LoadInt32(&d.status)
		switch state {
		case statusExit:
			return
		}

		select {
		case w := <-d.workersc:

			// validate worker is OK
			if w.Drain() {
				// drain resource
				continue
			}

			// take task, run task, return worker to pool
			// prevent block
			select {
			case job := <-d.tasksc:
				w.Work(job)
				d.wg.Done()
			default:
			}

			select {
			case d.workersc <- w:
			default:
				// log.Printf("run Can't return worker")
			}
		default:
		}
	}
}

// Populate adds a worker.
// if workers is equal to worker channel capacity Populate will
// duplicate workers channel size and respawn run method.
//
// Populate method must be used at init time. You can use it
// at runtime but take in mind that will stop all current tasks.
func (d *JobQ) Populate(w Worker) error {
	select {
	case d.workersc <- w:
		return nil
	default:
		return ErrAddWorker
	}

	return nil
}

// AddTask add a task to queue.
func (d *JobQ) AddTask(task TaskFunc) error {
	if atomic.LoadInt32(&d.status) == statusStopping {
		return ErrQuit
	}

	select {
	case d.tasksc <- task:
		d.wg.Add(1)
	default:
		return ErrQueueSizeExceeded
	}
	return nil
}

// Stop stops all workers and prevent tasks from be added to
// queue.
//
// All tasks added before calling Stop will complete.
func (d *JobQ) Stop() {
	// this will complete task from New call.
	if atomic.LoadInt32(&d.status) != statusStopping {
		atomic.StoreInt32(&d.status, statusStopping)
		d.wg.Done()
		log.Printf("Done")
	}
}

// Wait make JobQ waits until all tasks are done.
func (d *JobQ) Wait() {
	d.wg.Wait() // wait until Stop is call.

	if atomic.LoadInt32(&d.status) != statusExit {
		atomic.StoreInt32(&d.status, statusExit)
	}
}

// Worker interface defines behaviour for a worker.
type Worker interface {
	// Work runs a task and makes Worker return to pool.
	//
	// If a task timeouts then Work method will send an empty
	// struct (struct{}{}) to cancel channel inside TaskFunc.
	// This will allow users to return when a task fails and
	// prevent hangs.
	Work(TaskFunc) error

	// Drain method tell JobQ that your worker must be put
	// apart from workers.
	//
	// It's useful if you have a custom worker that has
	// expensive resources (like database connections) that
	// for some application logic needs to be removed from
	// pool.
	//
	// Remember that when a Drain method returns
	// true you will have less workers so a Populate method
	// call is required to replace the worker.
	Drain() bool
}
