package main

import (
	"errors"
	"flag"
	"log"
	"time"

	"github.com/jimmy-go/jobq"
)

var (
	maxWs = flag.Int("max-workers", 3, "Number of workers.")
	maxQs = flag.Int("max-queue", 10, "Number of queue works.")
	tasks = flag.Int("tasks", 20, "Number of tasks.")
)

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile)
	log.Printf("workers [%d]", *maxWs)
	log.Printf("queue len [%d]", *maxQs)
	log.Printf("tasks [%d]", *tasks)

	errc := make(chan error)
	go func() {
		for err := range errc {
			log.Printf("main : error channel : err [%s]", err)
		}
	}()

	q := make(chan jobq.Job, *maxQs)
	d := jobq.NewDispatcher(*maxWs, q, errc)
	d.Run()

	// declare new session.

	for i := 0; i < *tasks; i++ {
		func(index int) {
			// declare some task.
			task := func(ii int, e chan error) {
				t := time.Duration(time.Now().Second()/10) * time.Second
				log.Printf("main : sleep [%v]", t)
				time.Sleep(t)

				log.Printf("main : task [%d] done!", index)
				e <- errors.New("just some planned error")

				// for database operations you can declare a new session that
				// all tasks must share
			}
			q <- task
		}(i)
	}

	// select {}
	time.Sleep(1 * time.Minute)
	for {
		log.Println("sleep 1 minute!")
		time.Sleep(1 * time.Minute)
	}
}
