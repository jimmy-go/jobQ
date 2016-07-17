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

// TestTable tests invalid inputs.
func TestTable(t *testing.T) {
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
			// log.Printf("TestTable : err [%s] expected [%s]", err, x.M.Expected)
			t.FailNow()
			return
		}
	}
}

// TestWork it needs to be proved running.
func TestWork(t *testing.T) {
	size := 10

	jq, err := New(2, size, DefaultTimeout)
	if err != nil {
		t.FailNow()
		return
	}

	c := make(chan int, size)

	go func() {
		var count int
		for i := range c {
			log.Printf("TestWork : Stop All at [%v]", i)
			count++
			if count >= size {
				_ = i
				jq.Stop()
				log.Printf("TestWork : Stop All at [%v] return", i)
				return
			}
		}
	}()

	go func() {
		for i := 0; i < size; i++ {
			// change m with i and see a racecondition.
			go func(m int) {
				jq.AddTask(func(cancel chan struct{}) error {
					c <- m
					return nil
				})
			}(i)
		}

		jq.Stop()
	}()

	jq.Wait()
}

func bench(workers, queue int, b *testing.B) {

	// log.Printf("bench : workers [%v] queue [%v]", workers, queue)

	jq, err := New(workers, queue, DefaultTimeout)
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
