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
	maxWs = flag.Int("max-workers", 5, "Number of workers.")
	maxQs = flag.Int("max-queue", 10, "Number of queue works.")
	tasks = flag.Int("tasks", 200000, "Number of tasks.")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	log.SetFlags(log.Lshortfile)
	log.Printf("workers [%d]", *maxWs)
	log.Printf("queue len [%d]", *maxQs)
	log.Printf("tasks [%d]", *tasks)

	errc := make(chan error)
	go func() {
		for err := range errc {
			if err != nil {
				log.Printf("main : error channel : err [%s]", err)
			}
		}
	}()

	q := make(chan jobq.Job, *maxQs)
	d := jobq.NewDispatcher(*maxWs, q, errc)
	d.Run()

	// declare new session.
	go func() {
		select {
		case <-time.After(20 * time.Second):
			panic(errors.New("panic to see gorutines"))
		}
	}()

	for i := 0; i < *tasks; i++ {
		func(index int) {
			task := func() error {
				t := time.Duration(time.Now().Second()/10) * 1000 * time.Millisecond
				log.Printf("main : sleep [%v]", t)
				time.Sleep(t)
				log.Printf("main : task [%d] done!", index)
				return nil
			}
			select {
			case q <- task:
			}
		}(i)
	}

	for {
		log.Println("sleep 1 minute!")
		time.Sleep(1 * time.Minute)
	}
}
