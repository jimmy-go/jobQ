// Package jobq contains tools for a worker pool with queue limit.
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
	"sync"
	"time"
)

var (
	// DefaultJobQ it's the default JobQ dispatcher.
	// will be enough for several cases.
	DefaultJobQ, _ = New(2, 2, 250*time.Millisecond)

	// DefaultTimeout is default duration for tasks.
	DefaultTimeout = time.Duration(1 * time.Second)

	// ErrStopped error is returned when no more tasks are
	// allowed to run on JobQ.
	ErrStopped = errors.New("jobq: no more task allowed")

	// ErrFullWorkers error is returned when you call
	// Populate method and JobQ.size is exceded.
	ErrFullWorkers = errors.New("jobq: workers are full")

	// ErrTimeout error is returned when a task timeouts.
	ErrTimeout = errors.New("jobq: task timeout")

	// ErrInvalidWorkerSize error is returned when you init
	// a new JobQ with less than 1 worker(s).
	ErrInvalidWorkerSize = errors.New("jobq: invalid worker size")

	// ErrInvalidQueueSize error is returned when you init
	// a new JobQ with queue length less than 1.
	ErrInvalidQueueSize = errors.New("jobq: invalid queue size")
)

// TaskFunc defines behavior for a task.
type TaskFunc func(chan struct{}) error

// JobQ share jobs between available workers.
type JobQ struct {
	// workersc contain all workers from JobQ.
	// It blocks when there are not workers available.
	workersc chan Worker

	// tasksc contain all tasks to run, it locks JobQ when
	// is full. So use it carefully when designing your
	// system and DO MEASURES OF WHAT YOU NEED.
	tasksc chan TaskFunc

	// size is the number of workers for deploy, sets
	// workersc len and cap.
	size int

	// done send finalization signal to JobQ.
	done chan struct{}

	// exit is populated when Stop is called, so every
	// AddTask call after that will be skipped and no task
	// will be added to queue.
	exit chan struct{}

	// populate is full when Populate method is called and
	// does JobQ.run() quit.
	populate chan struct{}

	// wg defines how many tasks are running currently.
	// it locks on JobQ.Wait() method.
	wg sync.WaitGroup

	// lock is needed to know when JobQ is ending all his
	// tasks. It's unlocked by Stop method.
	lock sync.WaitGroup
}

// New returns a new JobQ dispatcher.
//
// size: how many workers would init.
// queueLen: how many tasks would put in queue before block.
// timeout: duration limit for every task.
func New(size int, queueLen int, timeout time.Duration) (*JobQ, error) {
	if size < 1 {
		return nil, ErrInvalidWorkerSize
	}
	if queueLen < 1 {
		return nil, ErrInvalidQueueSize
	}
	d := &JobQ{
		workersc: make(chan Worker, size),
		tasksc:   make(chan TaskFunc, queueLen),
		size:     size,
		done:     make(chan struct{}, 1),
		exit:     make(chan struct{}, 1),
		populate: make(chan struct{}, 1),
	}

	// lock JobQ in Wait until Stop is called.
	d.lock.Add(1)

	// populate workers
	for i := 0; i < d.size; i++ {
		w := newDefaultWorker(timeout)
		// every Populate call stop current JobQ.run() and restart it with another
		// goroutine
		err := d.Populate(w)
		if err != nil {
			return nil, err
		}
	}

	return d, nil
}

// run keep dispatching tasks between workers.
func (d *JobQ) run() {
	for {
		select {
		case w := <-d.workersc:
			go func() {
				job := <-d.tasksc
				w.Do(job)
				d.workersc <- w
				d.wg.Done()
			}()
		case <-d.done:

			// if exit is full means Stop has been called
			// so with can realase this lock.
			if len(d.tasksc) < 1 && len(d.exit) > 0 {
				// log.Printf("JobQ : run exit")
				d.lock.Done()
				return
			}

			// keep full is not ready yet to quit
			//
			// and don't forget to drain before
			drainempty(d.done)
			d.done <- struct{}{}
		case <-d.populate:
			if len(d.tasksc) < 1 {
				return
			}
			drainempty(d.populate)
			d.populate <- struct{}{}
		}
	}
}

// Populate adds a worker.
//
// every call to Populate stops current JobQ.run() method
// and respawn it.
func (d *JobQ) Populate(w Worker) error {

	// make quit previous run method.
	drainempty(d.populate)
	d.populate <- struct{}{}

	if len(d.workersc) >= d.size {
		return ErrFullWorkers
	}

	d.workersc <- w

	// start receiving tasks
	go d.run()

	return nil
}

// AddTask add a task (see TaskFunc) to queue.
func (d *JobQ) AddTask(task TaskFunc) error {
	if len(d.exit) > 0 {
		return ErrStopped
	}
	d.tasksc <- task
	d.wg.Add(1)
	return nil
}

// Stop stops all workers and prevent tasks from be added to
// queue.
//
// All tasks added before calling Stop will complete.
func (d *JobQ) Stop() {

	// prevent multiple calls to Stop, prevent buggy
	// behaviour.
	if len(d.exit) > 0 {
		return
	}

	drainempty(d.done)
	drainempty(d.exit)
	d.done <- struct{}{}
	d.exit <- struct{}{}
}

// Wait make JobQ waits until all tasks are done.
func (d *JobQ) Wait() {
	d.wg.Wait()   // wait until works are done.
	d.lock.Wait() // wait until Stop is call.

	// clean your mess honey, we don't leave traces.
	drainworkersc(d.workersc)
	draintasksc(d.tasksc)
}

// Worker interface
//
// Defines behaviour for a worker.
type Worker interface {

	// Do runs a task and makes Worker return to pool.
	//
	// If a task timeouts then Do method will send a empty
	// struct (struct{}{}) to cancel channel inside TaskFunc.
	// This will allow users to return when a task fails and
	// prevent hangs.
	Do(TaskFunc) error

	// Drain method tell JobQ that your worker must be put
	// apart from work.
	//
	// It's useful if you have a custom worker that has
	// expensive resources (like database connections) that
	// for some application logic needs to be removed from
	// pool.
	//
	// Remember that when a Worker.Drain() method returns
	// true you will have less workers so a Populate method
	// call is required to replace lossed worker.
	Drain() bool
}

// DefaultWorker implements Worker interface.
//
// Make a full picture of a Worker implementation.
type DefaultWorker struct {
	timeout time.Duration
	errc    chan error
	cancel  chan struct{}
}

// newDefaultWorker returns a new DefaultWorker.
func newDefaultWorker(timeout time.Duration) *DefaultWorker {
	w := &DefaultWorker{
		cancel:  make(chan struct{}, 1),
		errc:    make(chan error, 1),
		timeout: timeout,
	}
	return w
}

// Do satisfies Worker interface.
func (w *DefaultWorker) Do(task TaskFunc) error {
	// drain channels before use
	drainempty(w.cancel)
	drainerrc(w.errc)

	go func() {
		err := task(w.cancel)
		w.errc <- err
	}()
	select {
	case err := <-w.errc:
		return err
	case <-time.After(w.timeout):
		w.cancel <- struct{}{}
		return ErrTimeout
	}
	return ErrTimeout
}

// Drain implements Worker interface.
//
// enables JobQ to remove this worker from pool
// of workers.
func (w *DefaultWorker) Drain() bool {
	// this worker don't need to be drained.
	return false
}

// drainerrc empty error channel.
func drainerrc(c chan error) {
	for i := 0; i < len(c); i++ {
		<-c
	}
}

// drainempty drains empty struct{} channel.
func drainempty(c chan struct{}) {
	for i := 0; i < len(c); i++ {
		<-c
	}
}

// draintasksc empty TaskFunc channel.
func draintasksc(c chan TaskFunc) {
	for i := 0; i < len(c); i++ {
		<-c
	}
}

// drainworkersc empty Worker channel.
func drainworkersc(c chan Worker) {
	for i := 0; i < len(c); i++ {
		<-c
	}
}
