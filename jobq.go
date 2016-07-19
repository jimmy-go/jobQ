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
	"log"
	"sync"
	"time"
)

var (
	// DefaultJobQ it's the default JobQ dispatcher.
	// will be enough for several cases.
	// DefaultJobQ, _ = New(2, 2, 250*time.Millisecond)

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

	// ErrWorkersNotFound _
	ErrWorkersNotFound = errors.New("jobq: workers not found, channel is empty?")
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
	size int32

	// lock is needed to know when JobQ is ending all his
	// tasks. It's unlocked by Stop method.
	lock sync.WaitGroup

	// done is full when Stop method is called.
	done chan struct{}

	block chan struct{}

	mut sync.RWMutex
}

// NewDefault returns a new JobQ dispatcher with worker
// type DefaultWorker.
//
// In most cases this will be enough for your requisites,
// but take a look on Worker interface first.
//
// size: how many workers would init.
// queueLen: how many tasks can take before block.
// timeout: duration limit for task.
//
func NewDefault(size, queueLen int, timeout time.Duration) (*JobQ, error) {
	d, err := New(size, queueLen, timeout)
	if err != nil {
		return nil, err
	}

	// populate workers
	for i := 0; i < size; i++ {
		w := newDefaultWorker(timeout)
		err := d.Populate(w)
		if err != nil {
			return nil, err
		}
	}

	return d, nil
}

// Must returns a new JobQ or panics.
//
func Must(size, queueLen int, timeout time.Duration) *JobQ {
	d, err := New(size, queueLen, timeout)
	if err != nil {
		panic(err)
	}
	return d
}

// New generates a empty JobQ.
//
// workers indicate cap of channel workersc before grow.
// queueLen indicate cap of queue channel before block.
// You need to call Populate method after calling New
//
func New(workers, queueLen int, timeout time.Duration) (*JobQ, error) {
	if workers < 1 {
		return nil, ErrInvalidWorkerSize
	}
	if queueLen < 1 {
		return nil, ErrInvalidQueueSize
	}
	d := &JobQ{
		tasksc:   make(chan TaskFunc, queueLen),
		workersc: make(chan Worker, workers*2),
		done:     make(chan struct{}, 1),
		block:    make(chan struct{}, 1),
	}

	// this will be unlocked by Stop method.
	log.Printf("JobQ : New : lock add 1")
	d.lock.Add(1)

	// start receiving tasks
	go d.run()

	return d, nil
}

// run keep dispatching tasks between workers.
func (d *JobQ) run() {
	log.Printf("JobQ : run start")

	// lock JobQ in Wait until Stop is called.
	log.Printf("JobQ : run : lock add 1")
	d.lock.Add(1)
	defer func() {
		log.Printf("JobQ : run return")
		log.Printf("JobQ : run : lock done 1")
		d.lock.Done()
		d.block <- struct{}{}
	}()

	for {
		if len(d.workersc) < 1 {
			if len(d.done) > 0 && len(d.tasksc) < 1 {
				return
			}
			continue
		}
		w := <-d.workersc

		// if done is full means Stop has been called
		// so can realase this lock.
		if len(d.done) > 0 && len(d.tasksc) < 1 {
			return
		}

		// validate worker is OK
		if w.Drain() {
			// drain resource
			continue
		}

		// take task, run task, return worker to pool
		// prevent block
		if len(d.tasksc) > 0 {
			job := <-d.tasksc
			w.Do(job)
		}
		d.workersc <- w
	}
}

// Populate adds a worker.
//
// if workers is equal to len(workersc) this will duplicate
// workersc size and respawn JobQ.run method.
//
func (d *JobQ) Populate(w Worker) error {
	log.Printf("JobQ : Populate : start")
	d.mut.RLock()
	defer func() {
		d.mut.RUnlock()
		log.Printf("JobQ : Populate : return")
	}()

	maxcap := cap(d.workersc)
	if len(d.workersc) < maxcap {
		d.workersc <- w
		return nil
	}

	log.Printf("JobQ : Populate : size [%v] cap [%v]", len(d.workersc), maxcap)
	if len(d.workersc) < 1 {
		return ErrWorkersNotFound
	}

	// stop JobQ.run
	drainempty(d.done)
	d.done <- struct{}{}

	// block until run exits
	<-d.block
	log.Printf("JobQ : Populate : unblocked")

	// make cache from tasks to prevent block.
	cachetask := make(chan TaskFunc, 1000*1000)
	go func() {
		if len(d.tasksc) < 1 {
			log.Printf("JobQ : Populate : cached tasks : return")
			return
		}
		for task := range d.tasksc {
			log.Printf("JobQ : Populate : cached tasks : take task")
			if len(d.tasksc) < 2 {
				log.Printf("JobQ : Populate : cached tasks")
				return
			}
			cachetask <- task
		}
	}()

	// keep workers
	cache := make(chan Worker, maxcap)
	if len(d.workersc) > 0 {
		for i := 0; i < len(d.workersc); i++ {
			x := <-d.workersc
			cache <- x
		}
	}

	log.Printf("JobQ : Populate : workers cached")

	// repopulate worker channel.
	d.workersc = make(chan Worker, maxcap*2)
	if len(cache) > 0 {
		for i := 0; i < len(cache); i++ {
			y := <-cache
			d.workersc <- y
		}
	}

	d.workersc <- w

	log.Printf("JobQ : Populate : workers restore")

	// start receiving tasks again
	go d.run()

	// restore cache
	go func() {
		if len(cachetask) < 1 {
			return
		}
		for task := range cachetask {
			if len(cachetask) < 3 {
				log.Printf("JobQ : Populate : cache tasks repopulated")
				return
			}
			d.tasksc <- task
		}
	}()

	return nil
}

// AddTask add a task (see TaskFunc) to queue.
func (d *JobQ) AddTask(task TaskFunc) error {
	if len(d.workersc) < 1 {
		return ErrWorkersNotFound
	}
	if len(d.done) > 0 {
		return ErrStopped
	}
	d.tasksc <- task
	return nil
}

// Stop stops all workers and prevent tasks from be added to
// queue.
//
// All tasks added before calling Stop will complete.
func (d *JobQ) Stop() {
	log.Printf("JobQ : Stop : start")
	defer log.Printf("JobQ : Stop : return")

	// prevent multiple calls
	if len(d.done) > 0 {
		return
	}

	// this will complete task from New call.
	log.Printf("JobQ : Stop : lock done 1")
	d.lock.Done()

	drainempty(d.done)
	d.done <- struct{}{}

	w := <-d.workersc
	d.workersc <- w
}

// Wait make JobQ waits until all tasks are done.
func (d *JobQ) Wait() {
	d.lock.Wait() // wait until Stop is call.
	log.Printf("JobQ : Wait : wait complete")

	// clean your mess honey, we don't leave traces.
	drainworkersc(d.workersc)
	draintasksc(d.tasksc)
	drainempty(d.done)
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
