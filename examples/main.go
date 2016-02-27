package main

import (
	"errors"
	"flag"
	"log"
	"runtime"
	"time"

	"github.com/jimmy-go/jobq"
)

var (
	ws    = flag.Int("workers", 5, "Number of workers.")
	qlen  = flag.Int("queue", 10, "Number of queue works.")
	tasks = flag.Int("tasks", 100, "Number of tasks.")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	log.Printf("workers [%d]", *ws)
	log.Printf("queue len [%d]", *qlen)
	log.Printf("tasks [%d]", *tasks)
	w := 650 * time.Millisecond
	log.Printf("time for mock task [%v]", w)

	done := make(chan struct{}, 1)
	// want to see how many goroutines are running.
	go func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				log.Printf("main : GOROUTINES [%v]", runtime.NumGoroutine())
			case <-done:
				log.Printf("done!")
				return
			}
		}
	}()

	// declare a new worker pool

	// ws: workers size count.
	// qlen: size for queue length. All left jobs will wait until queue release some slot.
	jq, err := jobq.New(*ws, *qlen)
	if err != nil {
		log.Printf("main : err [%s]", err)
	}

	for i := 0; i < *tasks; i++ {
		go func(index int) {
			// task satisfies type jobq.Job, can be any function with error return.
			// I take this aproximation because some worker pool I see around use an interface
			// what I consider a limiting factor.
			// this way you only need declare a function and you're ready to go!
			// [if you think I'm doing it wrong please tell me :)]
			task := func() error {
				time.Sleep(w)
				log.Printf("main : task [%d] done!", index)
				return nil
			}
			// send the job to the queue.
			jq.Add(task)
		}(i)
	}

	log.Println("sleep 15 second!")
	time.Sleep(15 * time.Second)
	jq.Stop()
	done <- struct{}{}
	time.Sleep(15 * time.Second)
	panic(errors.New("see goroutines"))
}
