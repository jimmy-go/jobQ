package main

import (
	"log"
	"time"

	"github.com/cedmundo/handyman"
	"github.com/goinggo/work"
	"github.com/jimmy-go/jobq"
)

// W satisfies Work interface.
type W struct {
	C chan int
}

// Work func.
func (w *W) Work(id int) {
	<-time.After(25 * time.Millisecond)
	w.C <- id
}

// H satisfies handyman interface.
type H struct {
	C  chan int
	ID int
}

// Run func.
func (h *H) Run() {
	<-time.After(25 * time.Millisecond)
	h.C <- h.ID
}

func measure(who string, c chan int, start time.Time) {
	for i := range c {
		if i > 98 {
			log.Printf("%s : T [%s] j [%v]", who, time.Since(start), i)
			return
		}
	}
}

func main() {
	// goinggo.Work
	go func() {
		c := make(chan int, 100)
		go measure("Work", c, time.Now())
		p, _ := work.New(10, 15*time.Second, func(string) {})
		for i := 0; i < 100; i++ {
			go func(x int) {
				p.Run(&W{
					C: c,
				})
			}(i)
		}
		// p.Shutdown()
	}()
	// handyman
	go func() {
		c := make(chan int, 100)
		go measure("Handyman", c, time.Now())
		pool := handyman.NewPool()
		go pool.Monitor(10)
		for i := 0; i < 100; i++ {
			go func(x int) {
				pool.Queue <- &H{
					C:  c,
					ID: x,
				}
			}(i)
		}
		// pool.Close()
	}()
	// jobq
	go func() {
		c := make(chan int, 100)
		go measure("JoqQ", c, time.Now())
		jq, _ := jobq.New(10, 10)
		for i := 0; i < 100; i++ {
			go func(x int) {
				task := func() error {
					<-time.After(25 * time.Millisecond)
					c <- x
					return nil
				}
				jq.Add(task)
			}(i)
		}
		// jq.Stop()
	}()

	select {
	case <-time.After(1 * time.Second):
		log.Printf("Exit")
	}
}
