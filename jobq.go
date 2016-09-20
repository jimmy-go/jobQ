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
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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

	// ErrQueueFull _
	ErrQueueFull = errors.New("jobq: queue is full")
)

const (
	statusEmpty = int32(iota + 1)
	statusRunning
	statusExit
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

	mut sync.RWMutex

	status int32

	block chan struct{}
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
		status:   statusEmpty,
		block:    make(chan struct{}, 1),
	}

	// this will be unlocked by Stop method.
	printf("JobQ : New : lock add 1")
	d.lock.Add(1)

	// start receiving tasks
	go d.run()

	return d, nil
}

// run keep dispatching tasks between workers.
func (d *JobQ) run() {
	printf("JobQ : run start")

	// lock JobQ in Wait until Stop is called.
	printf("JobQ : run : lock add 1")
	d.lock.Add(1)

	// make sure change once
	if atomic.LoadInt32(&d.status) == statusEmpty {
		atomic.StoreInt32(&d.status, statusRunning)
	}

	defer func() {
		printf("JobQ : run : before lock done 1")
		d.lock.Done()
		printf("JobQ : run : lock done 1")
		printf("JobQ : run : before fill block chan")
		d.block <- struct{}{}
		printf("JobQ : run : fill block chan")
		printf("JobQ : run : return")
	}()

	for {
		printf("JobQ : run : for begin")
		// if len(d.done) > 0 && len(d.tasksc) < 1 {
		if len(d.tasksc) < 1 && atomic.LoadInt32(&d.status) == statusExit {
			printf("JobQ : run : no more tasks and exit status")
			return
		}
		printf("JobQ : run : validated exit")
		if len(d.workersc) < 1 {
			printf("JobQ : run : zero workers")
			continue
		}
		printf("JobQ : run : len before take it [%v]", len(d.workersc))
		select {
		default:
			printf("JobQ : run : can't return worker to pool")
		case w := <-d.workersc:
			printf("JobQ : run : worker return to pool")

			// validate worker is OK
			if w.Drain() {
				// drain resource
				printf("JobQ : run : drain worker")
				continue
			}

			// take task, run task, return worker to pool
			// prevent block
			if len(d.tasksc) > 0 {
				printf("JobQ : run : before take job")
				select {
				default:
					printf("JobQ : run : can't take job")
				case job := <-d.tasksc:
					printf("JobQ : run : take job")
					err := w.Do(job)
					if err != nil {
						printf("JobQ : run : job : err [%s]", err)
					}
					printf("JobQ : run : job done")
				}
			}
			printf("JobQ : run : before return worker")
			select {
			default:
				printf("JobQ : run : can't return worker")
				continue
			case d.workersc <- w:
			}
			printf("JobQ : run : return worker")
		}

		printf("JobQ : run : before return worker : len [%v] cap [%v]", len(d.workersc), cap(d.workersc))
		if len(d.workersc) == cap(d.workersc) {
			// FIXME; when populate is called multiple
			// times some workers can be drained because
			// of full channel
			printf("JobQ : run : drain worker, populate working in parallel?")

			//			go func() {
			//				// this maybe will run forever.
			//				for {
			//					if len(d.workersc) <= cap(d.workersc)+2 {
			//						d.workersc <- w
			//						printf("JobQ : run : worker is again in pool! len [%v] cap[%v]",
			//							len(d.workersc), cap(d.workersc))
			//						return
			//					}
			//				}
			//			}()
			continue
		}
	}
}

// Populate adds a worker.
//
// if workers is equal to len(workersc) this will duplicate
// workersc size and respawn JobQ.run method.
// Populate it's a expensive resource, don't call it so
// frecuently
//
func (d *JobQ) Populate(w Worker) error {
	printf("JobQ : Populate : start")
	d.mut.RLock()
	defer func() {
		d.mut.RUnlock()
		printf("JobQ : Populate : size [%v] cap [%v]", len(d.workersc), cap(d.workersc))
		printf("JobQ : Populate : return")
	}()

	maxcap := cap(d.workersc)
	if len(d.workersc)+1 < maxcap {
		d.workersc <- w
		return nil
	}

	// stop JobQ.run
	printf("JobQ : Populate : before make run quit")
	atomic.StoreInt32(&d.status, statusExit)
	printf("JobQ : Populate : make run quit")

	<-d.block

	// make cache from tasks to prevent block.
	cachetask := make(chan TaskFunc, 1000*1000)
	go func() {
		if len(d.tasksc) < 1 {
			printf("JobQ : Populate : cached tasks : return")
			return
		}
		for task := range d.tasksc {
			printf("JobQ : Populate : cached tasks : take task")
			if len(d.tasksc) < 2 {
				printf("JobQ : Populate : cached tasks")
				return
			}
			cachetask <- task
		}
	}()

	// keep workers
	cache := make(chan Worker, maxcap)
	if len(d.workersc) > 0 {
		for i := 0; i < len(d.workersc); i++ {
			go func() {
				x := <-d.workersc
				cache <- x
			}()
		}
	}

	printf("JobQ : Populate : workers cached")

	// start receiving tasks again
	go d.run()

	// repopulate worker channel.
	d.workersc = make(chan Worker, maxcap*2)
	if len(cache) > 0 {
		for i := 0; i < len(cache); i++ {
			go func() {
				y := <-cache
				d.workersc <- y
			}()
		}
	}

	printf("JobQ : Populate : before workers restore")
	select {
	default:
		printf("JobQ : Populate : fail")
		return errors.New("jobq: can't add worker")
	case d.workersc <- w:
	}
	printf("JobQ : Populate : workers restore")

	// restore cache
	go func() {
		if len(cachetask) < 1 {
			return
		}
		for task := range cachetask {
			if len(cachetask) < 3 {
				printf("JobQ : Populate : cache tasks repopulated")
				return
			}
			d.tasksc <- task
		}
	}()

	return nil
}

// AddTask add a task (see TaskFunc) to queue.
func (d *JobQ) AddTask(task TaskFunc) error {
	printf("JobQ : AddTask : start")
	// printf("JobQ : AddTask : queue len [%v] cap [%v]", len(d.tasksc), cap(d.tasksc))
	defer printf("JobQ : AddTask : return")

	printf("JobQ : AddTask : before validate status")
	// validate is running
	if atomic.LoadInt32(&d.status) != statusRunning {
		printf("JobQ : AddTask : jobq stopped")
		return ErrStopped
	}
	printf("JobQ : AddTask : validate status")

	printf("JobQ : AddTask : before send task : len [%v] cap [%v]", len(d.tasksc), cap(d.tasksc))
	select {
	default:
		printf("JobQ : AddTask : fail send task")
		return errors.New("jobq: add task err")
	case d.tasksc <- task:
		printf("JobQ : AddTask : task send it")
	}
	printf("JobQ : AddTask : send task")
	return nil
}

// Stop stops all workers and prevent tasks from be added to
// queue.
//
// All tasks added before calling Stop will complete.
func (d *JobQ) Stop() {
	printf("JobQ : Stop : start")
	defer printf("JobQ : Stop : return")

	// prevent multiple calls
	// if len(d.done) > 0 {
	if atomic.LoadInt32(&d.status) == statusExit {
		printf("JobQ : Stop : prevent multiple calls")
		return
	}
	atomic.StoreInt32(&d.status, statusExit)

	// this will complete task from New call.
	printf("JobQ : Stop : lock done 1")
	d.lock.Done()

	//	drainempty(d.done)
	//	d.done <- struct{}{}

	if len(d.workersc) > 0 {
		w := <-d.workersc
		d.workersc <- w
	}
}

// Wait make JobQ waits until all tasks are done.
func (d *JobQ) Wait() {
	d.lock.Wait() // wait until Stop is call.
	printf("JobQ : Wait : wait complete")

	//	// clean your mess honey, we don't leave traces.
	//	drainworkersc(d.workersc)
	//	draintasksc(d.tasksc)
	//	drainempty(d.done)
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

// printf logs useful information about JobQ operation.
func printf(s string, args ...interface{}) {
	_, f, l, _ := runtime.Caller(1)
	x := strings.Split(filepath.Clean(f), "/")
	name := x[len(x)-1:]
	log.Println(fmt.Sprintf("%v:%v", name, l), fmt.Sprintf(s, args...))
}
