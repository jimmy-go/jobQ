package jobq

import (
	"log"
	"testing"
	"time"
)

// TestNew tests invalid inputs.
func TestNew(t *testing.T) {
	type T struct {
		Size     int
		Len      int
		Expected error
	}
	var tests = []T{
		{1, 2, nil},
		{7, 6, nil},
		{1, -1, errInvalidQueueSize},
		{-1, 1, errInvalidWorkerSize},
	}
	for _, m := range tests {
		func(mo T) {
			_, err := New(mo.Size, mo.Len)
			if err != mo.Expected {
				t.Fail()
			}
		}(m)
	}
}

// TestWork it needs to be proved running.
func TestWork(t *testing.T) {
	jq, err := New(1, 1)
	if err != nil {
		t.Fail()
	}
	c := make(chan int)
	go func() {
		for i := range c {
			if i > 100 {
				log.Printf("TestWork : fail i [%v]", i)
				t.Fail()
			}
		}
	}()
	for i := 0; i < 100; i++ {
		// change m with i and see a racecondition.
		m := i
		jq.Add(func() error {
			select {
			case c <- m:
			}
			return nil
		})
	}
	jq.Stop()
	<-time.After(25 * time.Millisecond)
}

func bench(workers, queue int, b *testing.B) {
	jq, err := New(workers, queue)
	if err != nil {
		panic(err)
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			task := func() error {
				<-time.After(10 * time.Millisecond)
				return nil
			}
			jq.Add(task)
		}
	})
	jq.Stop()
}

func Benchmark100x100(b *testing.B)   { bench(100, 100, b) }
func Benchmark10x100(b *testing.B)    { bench(10, 100, b) }
func Benchmark1x1(b *testing.B)       { bench(1, 1, b) }
func Benchmark1x2(b *testing.B)       { bench(1, 2, b) }
func Benchmark1x3(b *testing.B)       { bench(1, 3, b) }
func Benchmark1000x1000(b *testing.B) { bench(1000, 1000, b) }
func Benchmark10x30(b *testing.B)     { bench(10, 30, b) }
func Benchmark1x100(b *testing.B)     { bench(1, 100, b) }
