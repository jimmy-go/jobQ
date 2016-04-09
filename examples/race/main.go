package main

import (
	"log"
	"time"

	"github.com/cedmundo/handyman"
	"github.com/goinggo/work"
	"github.com/jimmy-go/jobq"
)

// W satisfies Work interface.
type W struct{}

// Work func.
func (w *W) Work(id int) {
	<-time.After(25 * time.Millisecond)
}

// H satisfies handyman interface.
type H struct{}

// Run func.
func (h *H) Run() {
	<-time.After(25 * time.Millisecond)
}

func main() {
	// handyman
	go func() {
		now := time.Now()
		pool := handyman.NewPool()
		go pool.Monitor(10)
		for i := 0; i < 100; i++ {
			go func() {
				pool.Queue <- &H{}
			}()
		}
		pool.Close()
		log.Printf("Handyman : T [%s]", time.Since(now))
	}()
	// goinggo.Work
	go func() {
		now := time.Now()
		p, _ := work.New(10, 15*time.Second, func(string) {})
		for i := 0; i < 100; i++ {
			go func() {
				p.Run(&W{})
			}()
		}
		log.Printf("Work : T [%s]", time.Since(now))
	}()
	// jobq
	go func() {
		now := time.Now()
		jq, _ := jobq.New(10, 10)
		for i := 0; i < 100; i++ {
			go func() {
				task := func() error {
					<-time.After(25 * time.Millisecond)
					return nil
				}
				jq.Add(task)
			}()
		}
		log.Printf("JobQ : T [%s]", time.Since(now))
	}()

	select {
	case <-time.After(2 * time.Second):
	}
}
