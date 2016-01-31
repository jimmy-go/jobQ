####Job Queue in Go

Simple but powerful job queue package in go.

Example:
```go
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
	log.SetFlags(log.Lshortfile | log.Ltime)

	errc := make(chan error)
	go func() {
		for err := range errc {
			log.Printf("main : channel errors : err [%s]", err)
		}
	}()

	q := make(chan jobq.Job, *maxQs)
	d := jobq.NewDispatcher(*maxWs, q, errc)
	d.Run()

	for i := 0; i < *tasks; i++ {
		// generate gorutine for visualization in log, in production never to this,
		// it will cause a gorutine leak.
		go func(index int) {
			// declare some task.
			q <- func(ii int, e chan error) {
				t := time.Duration(time.Now().Second()) * time.Second
				log.Printf("main : sleep [%v]", t)
				time.Sleep(t)

				log.Printf("main : func index done [%d]", index)
				e <- errors.New("Forced error")
			}
		}(i)
	}

	log.Println("waiting")
	// block forever
	select {}
}
```

##### To Do
- [ ] Fix deadlock.
- [ ] Tests.
- [x] Examples.