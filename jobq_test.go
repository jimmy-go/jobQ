package jobq

import (
	"log"
	"testing"
	"time"
)

func init() {
	// log.SetFlags(log.Lshortfile)
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
	var tests = []T{
		{1, 2, nil, DefaultTimeout},
		{7, 6, nil, DefaultTimeout},
		{1, -1, ErrInvalidQueueSize, DefaultTimeout},
		{-1, 1, ErrInvalidWorkerSize, DefaultTimeout},
	}
	for _, m := range tests {
		x := struct{ M T }{m}

		_, err := NewDefault(x.M.Size, x.M.Len, x.M.timeout)
		if err == nil && x.M.Expected == nil {
			continue
		}

		if err != nil && err.Error() != x.M.Expected.Error() {
			// log.Printf("TestTable : err [%s] expected [%s]", err, x.M.Expected)
			t.FailNow()
			return
		}
	}
}

// TestSimple create a new JobQ and make simple tasks until
// Stop is call.
func TestSimple(t *testing.T) {
	queue := 5

	jq, err := NewDefault(3, queue, DefaultTimeout)
	if err != nil {
		t.FailNow()
		return
	}

	c := make(chan int, queue)

	go func() {
		var count int
		for i := range c {
			// log.Printf("TestSimple : count [%v] i [%v]", count, i)
			count++
			if count >= queue {
				_ = i
				jq.Stop()
				// log.Printf("TestSimple : Stop at i [%v] count [%v] return", i, count)
				return
			}
		}
	}()

	go func() {
		for i := 0; i < queue; i++ {
			// change m with i and see a racecondition.
			go func(m int) {
				err := jq.AddTask(func(cancel chan struct{}) error {
					// log.Printf("TestSimple : send c [%v]", m)
					c <- m
					return nil
				})
				if err != nil {
					// log.Printf("TestSimple : task err [%s]", err)
				}
			}(i)
		}
	}()

	jq.Wait()
}

// TestPopulate demonstrates that you can keep adding tasks
// and regrow workers channel without issues.
func TestPopulate(t *testing.T) {
	jq, err := NewDefault(3, 10, DefaultTimeout)
	if err != nil {
		t.FailNow()
		return
	}

	var exit bool

	go func() {
		for {
			if exit {
				log.Printf("TestPopulate : add tasks exit")
				return
			}

			// log.Printf("TestPopulate : add task")
			err := jq.AddTask(func(cancel chan struct{}) error {
				// log.Printf("TestPopulate : task done")
				return nil
			})
			if err != nil {
				// log.Printf("TestPopulate : add task : err [%s]", err)
			}
		}
	}()
	go func() {
		for {
			if exit {
				log.Printf("TestPopulate : populate exit")
				return
			}

			w := newDefaultWorker(DefaultTimeout)
			err := jq.Populate(w)
			if err != nil {
				log.Printf("TestPopulate : Populate : err [%s]", err)
			}
		}
	}()

	<-time.After(10 * time.Millisecond)
	exit = true
	jq.Stop()

	jq.Wait()
}

// TestTimeout demonstrates JobQ can't block when task didn't
// return
func TestTimeout(t *testing.T) {
}

// TestCustomWorker makes work with a custom worker.
func TestCustomWorker(t *testing.T) {
}

func bench(workers, queue int, b *testing.B) {

	// log.Printf("bench : workers [%v] queue [%v]", workers, queue)

	jq, err := NewDefault(workers, queue, DefaultTimeout)
	if err != nil {
		b.Fail()
	}

	c := make(chan int, queue)
	go func() {
		var count int
		for i := range c {
			// log.Printf("bench : at [%v]", i)
			count++
			if count == queue {
				// log.Printf("bench : Stop All at [%v]", i)
				_ = i
				jq.Stop()
			}
		}
	}()

	b.RunParallel(func(pb *testing.PB) {
		var ii int
		for pb.Next() {
			ii++
			// log.Printf("bench : add")
			task := func(cancel chan struct{}) error {
				// c <- ii
				// log.Printf("bench : added")
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
