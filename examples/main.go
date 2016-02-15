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
	ws    = flag.Int("max-workers", 5, "Number of workers.")
	qlen  = flag.Int("max-queue", 10, "Number of queue works.")
	tasks = flag.Int("tasks", 100, "Number of tasks.")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	log.SetFlags(log.Lshortfile)
	log.Printf("workers [%d]", *ws)
	log.Printf("queue len [%d]", *qlen)
	log.Printf("tasks [%d]", *tasks)

	go func() {
		for {
			log.Printf("main : GOROUTINES [%v]", runtime.NumGoroutine())
			time.Sleep(5 * time.Second)
		}
	}()

	errc := make(chan error)
	go func() {
		for err := range errc {
			if err != nil {
				log.Printf("main : error channel : err [%s]", err)
			}
		}
	}()

	jq, err := jobq.New(*ws, *qlen, errc)
	if err != nil {
		log.Printf("main : err [%s]", err)
	}

	for i := 0; i < *tasks; i++ {
		func(index int) {
			task := func() error {
				time.Sleep(350 * time.Millisecond)
				log.Printf("main : task [%d] done!", index)
				return nil
			}
			jq.Add(task)
		}(i)
	}

	log.Println("sleep 15 second!")
	time.Sleep(15 * time.Second)
	jq.Stop()
	time.Sleep(15 * time.Second)
	panic(errors.New("see goroutines"))
}
