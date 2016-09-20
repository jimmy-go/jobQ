package main

import (
	"flag"
	"log"
	"runtime"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/jimmy-go/jobq"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	log.SetFlags(0)

	go func() {
		log.Printf("listen 6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	jq, err := jobq.NewDefault(3, 10, jobq.DefaultTimeout)
	if err != nil {
		log.Printf("err [%s]", err)
		return
	}

	var exit bool

	go func() {
		for {
			if exit {
				goru := runtime.NumGoroutine()
				log.Printf("TestPopulate : goroutines [%v]", goru)
				return
			}
		}
	}()
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

			w := newDefaultWorker(jobq.DefaultTimeout)
			err := jq.Populate(w)
			if err != nil {
				log.Printf("TestPopulate : Populate : err [%s]", err)
			}
		}
	}()

	log.Printf("TestPopulate : before wait time")
	time.Sleep(1500 * time.Millisecond)
	log.Printf("TestPopulate : wait time")
	exit = true
	jq.Stop()
	log.Printf("TestPopulate : between Stop & Wait")
	jq.Wait()
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
func (w *DefaultWorker) Do(task jobq.TaskFunc) error {
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
		return jobq.ErrTimeout
	}
	return jobq.ErrTimeout
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
