package jobq

import (
	"testing"
	"time"
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
		_, err := New(m.Size, m.Len)
		if err != m.Expected {
			t.Fail()
		}
	}
}

// TestWork it needs to be proved running.
func TestWork(t *testing.T) {
	jq, err := New(10, 20)
	if err != nil {
		t.Fail()
	}
	jq.Add(func() error {
		return nil
	})
	time.Sleep(time.Second)
	jq.Stop()
}
