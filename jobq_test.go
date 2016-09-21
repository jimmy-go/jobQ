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
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	_ "gopkg.in/jimmy-go/vovo.v0/profiling/defpprof"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFlags(log.Lshortfile)
}

// MyWorker implements Worker interface.
type MyWorker struct {
	drain bool
}

func (w *MyWorker) Work(task TaskFunc) error {

	cancel := make(chan struct{}, 100)
	errc := make(chan error, 2)
	go func() {
		err := task(cancel)
		errc <- err
	}()

	select {
	case <-time.After(time.Second):
		return errors.New("timeout")
	case <-cancel:
		return errors.New("cancelation")
	case err := <-errc:
		return err
	}

	return nil
}

func (w *MyWorker) Drain() bool {
	return w.drain
}

// T type for test.
type T struct {
	Size     int
	Len      int
	Expected error
	timeout  time.Duration
}

// TestTable basic table test
func TestTable(t *testing.T) {
	t.Logf("start table")
	defer t.Logf("return table")

	var tests = []T{
		{1, 2, nil, DefaultTimeout},
		{7, 6, nil, DefaultTimeout},
		{1, -1, ErrInvalidQueueSize, DefaultTimeout},
		{-1, 1, ErrInvalidWorkerSize, DefaultTimeout},
	}
	for _, m := range tests {
		x := struct{ M T }{m}

		_, err := New(x.M.Size, x.M.Len, x.M.timeout)
		if err == nil && x.M.Expected == nil {
			continue
		}

		if err != nil && err.Error() != x.M.Expected.Error() {
			t.Errorf("actual [%s] expected [%s]", err, x.M.Expected)
			return
		}
	}
}

// TestSimple create a new JobQ and make simple tasks until
// Stop is call.
func TestSimple(t *testing.T) {

	go func() {
		<-time.After(60 * time.Second)
		panic(errors.New("simple panic"))
	}()

	queue := 5
	jq, err := New(3, queue, DefaultTimeout)
	if err != nil {
		t.Errorf("err [%s]", err)
		return
	}
	for i := 0; i < 3; i++ {
		jq.Populate(&MyWorker{false})
	}

	c := make(chan int)

	go func() {
		var count int32
		for _ = range c {
			atomic.AddInt32(&count, 1)

			x := atomic.LoadInt32(&count)
			log.Printf("x [%v]", x)

			if x >= 4 {
				log.Printf("STOP x [%v]", x)
				jq.Stop()
				jq.Stop() // make sure calling twice make no change
				return
			}
		}
	}()

	go func() {
		for i := 0; i < queue*10; i++ {
			x := i

			err := jq.AddTask(func(cancel chan struct{}) error {
				select {
				case c <- x:
					log.Printf("c <- x")
				default:
					cancel <- struct{}{}
					log.Printf("cancel [%v]", x)
				}
				return nil
			})
			if err != nil {
				log.Printf("task err [%s]", err)
			}
		}
	}()

	jq.Wait()
}

// TestPopulate demonstrates that you can keep adding tasks
// and regrow workers channel without issues.
func TestPopulate(t *testing.T) {
	t.Logf("populate start")
	defer t.Logf("populate return")

	wrs := 4
	q := 1000

	jq, err := New(wrs, q, DefaultTimeout)
	if err != nil {
		t.Errorf("err [%s]", err)
		return
	}
	for i := 0; i < wrs; i++ {
		jq.Populate(&MyWorker{false})
	}

	tasks := 15000000

	c := make(chan int)

	go func() {
		var count int32
		for _ = range c {
			atomic.AddInt32(&count, 1)

			x := atomic.LoadInt32(&count)
			// log.Printf("DONE : X [%v]", x)

			// keep working until 40% of tasks are done.
			if x >= int32(6000) {
				log.Printf("stopped in [%v]", x)
				jq.Stop()
				jq.Stop()
				return
			}
		}
	}()

	go func() {
		<-time.After(10 * time.Millisecond)
		log.Printf("begin populate")
		for i := 0; i < 500; i++ {
			jq.Populate(&MyWorker{false})
		}
		log.Printf("populate done")
	}()

	go func() {
		for i := 0; i < tasks; i++ {
			x := i
			err := jq.AddTask(func(cancel chan struct{}) error {
				select {
				case c <- x:
				default:
					cancel <- struct{}{}
				}

				return nil
			})
			if err != nil {
				// 	log.Printf("add task : err [%s]", err)
			}
		}
	}()
	go func() {
		for i := 0; i < tasks; i++ {
			x := i
			err := jq.AddTask(func(cancel chan struct{}) error {
				select {
				case c <- x:
				default:
					cancel <- struct{}{}
				}

				return nil
			})
			if err != nil {
				// 	log.Printf("add task : err [%s]", err)
			}
		}
	}()

	jq.Wait()
}

// TestMust demonstrates Must returns a valid JobQ
func TestMust(t *testing.T) {
	t.Logf("start")
	defer t.Logf("return")

	defer func() {
		if err := recover(); err != nil {
			t.Errorf("recover : err [%s]", err)
		}
	}()

	// must work
	m := Must(2, 2, time.Second)
	if m == nil {
		t.Fail()
	}
}

// TestMustFailSize demonstrates panic if worker count is
// invalid
func TestMustFailSize(t *testing.T) {
	t.Logf("start")
	defer t.Logf("return")

	defer func() {
		if err := recover(); err != nil {
			if err != ErrInvalidWorkerSize {
				t.Errorf("recover : expected [%s] actual [%s]", ErrInvalidWorkerSize, err)
			}
		}
	}()
	// must panic
	Must(0, 1, time.Second)
}

// TestMustFailQueue demonstrates panic if queue len is
// invalid
func TestMustFailQueue(t *testing.T) {
	t.Logf("start")
	defer t.Logf("return")

	defer func() {
		if err := recover(); err != nil {
			if err != ErrInvalidQueueSize {
				t.Errorf("recover : expected [%s] actual [%s]", ErrInvalidQueueSize, err)
			}
		}
	}()
	// must panic
	Must(1, 0, time.Second)
}

// TestTimeout demonstrates JobQ can't block when task didn't return
func TestTimeout(t *testing.T) {
	t.Logf("start")
	defer t.Logf("return")

	jq, err := New(1, 1, 100*time.Millisecond)
	if err != nil {
		t.Fail()
		return
	}
	jq.Populate(&MyWorker{false})

	// task 1
	if err := jq.AddTask(func(cancel chan struct{}) error {
		log.Printf("before block")
		select {}
		log.Printf("after block")
		t.Logf("this log it will no show!")
		return nil
	}); err != nil {
		t.Logf("err [%s]", err)
		return
	}

	jq.Stop()
	jq.Wait()
}

func bench(workers, queue int, b *testing.B) {
	jq, err := New(workers, queue, DefaultTimeout)
	if err != nil {
		b.Errorf("err [%s]", err)
		b.Fail()
	}
	for i := 0; i < workers; i++ {
		jq.Populate(&MyWorker{false})
	}

	jobs := int32(queue)
	c := make(chan int32, queue)

	var count int32

	go func() {
		for i := range c {
			atomic.AddInt32(&count, 1)
			if atomic.LoadInt32(&count) >= jobs {
				_ = i
				jq.Stop()
			}
		}
	}()

	b.RunParallel(func(pb *testing.PB) {
		var ii int32
		for pb.Next() {
			atomic.AddInt32(&ii, 1)

			task := func(cancel chan struct{}) error {
				c <- atomic.LoadInt32(&ii)
				return nil
			}
			jq.AddTask(task)
		}
	})

	jq.Wait()
}

func Benchmark100x100(b *testing.B)   { bench(100, 100, b) }
func Benchmark10x100(b *testing.B)    { bench(10, 100, b) }
func Benchmark1x1(b *testing.B)       { bench(1, 1, b) }
func Benchmark1x2(b *testing.B)       { bench(1, 2, b) }
func Benchmark1x3(b *testing.B)       { bench(1, 3, b) }
func Benchmark1000x1000(b *testing.B) { bench(1000, 1000, b) }
func Benchmark10x30(b *testing.B)     { bench(10, 30, b) }
func Benchmark1x100(b *testing.B)     { bench(1, 100, b) }
