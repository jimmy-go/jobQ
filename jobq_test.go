package jobq

import (
	"runtime"
	"testing"
	"time"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
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
	t.Logf("TestTable : start")
	defer t.Logf("TestTable : return")

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
			// t.Logf("TestTable : err [%s] expected [%s]", err, x.M.Expected)
			t.FailNow()
			return
		}
	}
}

// TestSimple create a new JobQ and make simple tasks until
// Stop is call.
func TestSimple(t *testing.T) {
	t.Logf("TestSimple : start")
	defer t.Logf("TestSimple : return")

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
			// t.Logf("TestSimple : count [%v] i [%v]", count, i)
			count++
			if count >= queue {
				_ = i
				jq.Stop()
				jq.Stop() // make sure calling twice make no
				// change
				// t.Logf("TestSimple : Stop at i [%v] count [%v] return", i, count)
				return
			}
		}
	}()

	go func() {
		for i := 0; i < queue; i++ {
			// change m with i and see a racecondition.
			go func(m int) {
				err := jq.AddTask(func(cancel chan struct{}) error {
					// t.Logf("TestSimple : send c [%v]", m)
					c <- m
					return nil
				})
				if err != nil {
					// t.Logf("TestSimple : task err [%s]", err)
				}
			}(i)
		}
	}()

	jq.Wait()
}

// TestPopulate demonstrates that you can keep adding tasks
// and regrow workers channel without issues.
func TestPopulate(t *testing.T) {
	t.Logf("TestPopulate : start")
	defer t.Logf("TestPopulate : return")

	jq, err := NewDefault(3, 10, DefaultTimeout)
	if err != nil {
		t.FailNow()
		return
	}

	l := 1500

	go func() {
		for i := 0; i < l; i++ {
			goru := runtime.NumGoroutine()
			t.Logf("TestPopulate : goroutines [%v]", goru)
			return
		}
	}()
	go func() {
		for i := 0; i < l; i++ {
			// t.Logf("TestPopulate : add task")
			err := jq.AddTask(func(cancel chan struct{}) error {
				// t.Logf("TestPopulate : task done")
				return nil
			})
			if err != nil {
				// t.Logf("TestPopulate : add task : err [%s]", err)
			}
		}
	}()
	go func() {
		<-time.After(50 * time.Millisecond)
		for i := 0; i < 5; i++ {
			w := newDefaultWorker(DefaultTimeout)
			err := jq.Populate(w)
			if err != nil {
				t.Logf("TestPopulate : Populate : err [%s]", err)
			}
		}
	}()

	t.Logf("TestPopulate : before wait time")
	time.Sleep(1500 * time.Millisecond)
	t.Logf("TestPopulate : wait time")
	jq.Stop()
	t.Logf("TestPopulate : between Stop & Wait")
	jq.Wait()
}

// TestMustOK demonstrates Must returns a valid JobQ
func TestMustOK(t *testing.T) {
	t.Logf("TestMustOK : start")
	defer t.Logf("TestMustOK : return")

	defer func() {
		if err := recover(); err != nil {
			t.Fail()
			t.Logf("TestMust : recover err : [%s]", err)
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
	t.Logf("TestMustFailSize : start")
	defer t.Logf("TestMustFailSize : return")

	defer func() {
		if err := recover(); err != nil {
			if err != ErrInvalidWorkerSize {
				t.Fail()
				t.Logf("TestMust : recover err : [%s]", err)
			}
		}
	}()
	// must panic
	Must(0, 1, time.Second)
}

// TestMustFailQueue demonstrates panic if queue len is
// invalid
func TestMustFailQueue(t *testing.T) {
	t.Logf("TestMustFailQueue : start")
	defer t.Logf("TestMustFailQueue : return")

	defer func() {
		if err := recover(); err != nil {
			if err != ErrInvalidQueueSize {
				t.Fail()
				t.Logf("TestMust : recover err : [%s]", err)
			}
		}
	}()
	// must panic
	Must(1, 0, time.Second)
}

// TestTimeout demonstrates JobQ can't block when task didn't
// return
func TestTimeout(t *testing.T) {
	t.Logf("TestTimeout : start")
	defer t.Logf("TestTimeout : return")

	jq, err := NewDefault(1, 1, 100*time.Millisecond)
	if err != nil {
		t.Fail()
		return
	}

	// task 1
	if err := jq.AddTask(func(cancel chan struct{}) error {
		select {}
		t.Logf("TestTimeout : this log it will no show!")
		return nil
	}); err != nil {
		t.Fail()
		t.Logf("TestCustomWorker : err : [%s]", err)
		return
	}

	jq.Stop()
	jq.Wait()
}

// MyWorker implements Worker interface.
type MyWorker struct {
	drain bool
}

func (w *MyWorker) Do(task TaskFunc) error {
	return nil
}

func (w *MyWorker) Drain() bool {
	return w.drain
}

// TestCustomWorker makes work with a custom worker.
func TestCustomWorker(t *testing.T) {
	t.Logf("TestCustomWorker : start")
	defer t.Logf("TestCustomWorker : return")

	jq, err := New(2, 2, time.Second)
	if err != nil {
		t.Fail()
		return
	}
	// task 1
	if err := jq.AddTask(func(cancel chan struct{}) error {
		return nil
	}); err != ErrWorkersNotFound {
		t.Fail()
		t.Logf("TestCustomWorker : err : [%s] expected [%s]", err, ErrWorkersNotFound)
		return
	}

	w := &MyWorker{drain: false}
	err = jq.Populate(w)
	if err != nil {
		t.Fail()
		t.Logf("TestCustomWorker : err : [%s]", err)
		return
	}

	// task 2
	if err := jq.AddTask(func(cancel chan struct{}) error {
		return nil
	}); err != nil {
		t.Fail()
		t.Logf("TestCustomWorker : err : [%s]", err)
		return
	}

	// make worker be drained
	w.drain = true

	// task 3 will return err not workers found because was
	// drained.
	if err := jq.AddTask(func(cancel chan struct{}) error {
		return nil
	}); err != nil {
		t.Logf("TestCustomWorker : err : [%s]", err)
	}

	jq.Stop()
	jq.Wait()
}

func bench(workers, queue int, b *testing.B) {

	// t.Logf("bench : workers [%v] queue [%v]", workers, queue)

	jq, err := NewDefault(workers, queue, DefaultTimeout)
	if err != nil {
		b.Fail()
	}

	c := make(chan int, queue)
	go func() {
		var count int
		for i := range c {
			// t.Logf("bench : at [%v]", i)
			count++
			if count == queue {
				// t.Logf("bench : Stop All at [%v]", i)
				_ = i
				jq.Stop()
			}
		}
	}()

	b.RunParallel(func(pb *testing.PB) {
		var ii int
		for pb.Next() {
			ii++
			// t.Logf("bench : add")
			task := func(cancel chan struct{}) error {
				// c <- ii
				// t.Logf("bench : added")
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
