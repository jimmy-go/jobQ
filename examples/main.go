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
	t := 650 * time.Millisecond
	log.Printf("time for mock task [%v]", t)

	// want to see how many goroutines are running.
	go goroutines()

	// ws: workers size count.
	// qlen: size for queue length. All left jobs will wait until queue release some slot.
	jq, err := jobq.New(*ws, *qlen)
	if err != nil {
		log.Printf("main : err [%s]", err)
	}

	go func() {
		for i := 0; i < *tasks; i++ {
			go func(index int) {
				// task satisfies type jobq.Job, can be any function with error return.
				// I take this aproximation because some worker pool I see around use an interface
				// what I consider a limiting factor.
				// this way you only need declare a function and you're ready to go!
				now := time.Now()
				task := func() error {
					<-time.After(t)
					log.Printf("main : task [%d] done! T [%s]", index, time.Since(now))
					return nil
				}
				// send the job to the queue.
				jq.Add(task)

				if index == *tasks-1 {
					log.Printf("queue complete. stopping")
					jq.Stop()
				}

			}(i)
		}
	}()

	jq.Wait()
	panic(errors.New("see goroutines"))
}

func goroutines() {
	for {
		select {
		case <-time.After(5 * time.Second):
			log.Printf("main : GOROUTINES [%v]", runtime.NumGoroutine())
		}
	}
}
