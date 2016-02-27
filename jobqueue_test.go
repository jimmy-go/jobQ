package jobq

import (
	"log"
	"testing"
)

// TestNew tests invalid inputs.
func TestNew(t *testing.T) {
	type T struct {
		Size     int
		Len      int
		Expected error
	}
	var tests = []T{
		{1, 2, nil},
		{7, 6, nil},
		{1, -1, errInvalidQueueSize},
		{-1, 1, errInvalidWorkerSize},
	}
	for _, m := range tests {
		func(mo T) {
			_, err := New(mo.Size, mo.Len)
			if err != mo.Expected {
				t.Fail()
			}
		}(m)
	}
}

// TestWork it needs to be proved running.
func TestWork(t *testing.T) {
	jq, err := New(1, 1)
	if err != nil {
		t.Fail()
	}
	c := make(chan int, 100)
	go func() {
		for i := range c {
			if i > 100 {
				log.Printf("TestWork : fail i [%v]", i)
				t.Fail()
			}
		}
	}()
	for i := 0; i < 100; i++ {
		jq.Add(func() error {
			c <- i
			return nil
		})
	}
	log.Printf("TestWork : end")
	jq.Stop()
}
