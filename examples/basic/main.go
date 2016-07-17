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
	tasks = flag.Int("tasks", 40, "Number of tasks.")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	log.Printf("workers [%d]", *ws)
	log.Printf("queue len [%d]", *qlen)
	log.Printf("tasks [%d]", *tasks)
	t := 600 * time.Millisecond
	log.Printf("time for mock task [%v]", t)

	// want to see how many goroutines are running.
	go goroutines()

	// ws: workers size count.
	// qlen: size for queue length. All left jobs will wait
	// until queue release some slot.
	// timeout: timeout for every task
	jq, err := jobq.New(*ws, *qlen, time.Duration(1*time.Second))
	if err != nil {
		log.Printf("main : err [%s]", err)
	}

	go func() {
		defer func() {
			log.Printf("added all tasks")
		}()
		for i := 0; i < *tasks; i++ {
			log.Printf("add [%v] task", i)
			go func(index int) {
				now := time.Now()
				task := func(cancel chan struct{}) error {
					<-time.After(t)
					log.Printf("index [%v] done! T [%s]", index, time.Since(now))
					return nil
				}
				// send the job to the queue.
				jq.AddTask(task)

				if index == *tasks-1 {
					log.Printf("stopping queue. index [%v]", index)
					jq.Stop()
					jq.Stop() // test multiple calls to Stop
					jq.AddTask(func(cancel chan struct{}) error {
						log.Printf("Try aditional task when JobQ is stopped")
						return nil
					})
				}
			}(i)
		}
	}()

	go func() {
		<-time.After(14 * time.Second)
		// this case must return error because Stop was call
		err := jq.AddTask(func(cancel chan struct{}) error {
			log.Printf("add after 14 s")
			return nil
		})
		if err != nil {
			log.Printf("AddTask : err [%s]", err)
		}
	}()
	// <-time.After(15 * time.Second)
	// log.Printf("15 seconds wait complete")

	jq.Wait()
	panic(errors.New("see goroutines"))
}

func goroutines() {
	for {
		select {
		case <-time.After(35 * time.Second):
			log.Printf("main : GOROUTINES [%v]", runtime.NumGoroutine())
		}
	}
}
