package jobq

import (
	"errors"
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
		errc := make(chan error, 1)
		_, err := New(m.Size, m.Len, errc)
		if err != m.Expected {
			t.Fail()
		}
	}
}

// TestWork it needs to be proved running.
func TestWork(t *testing.T) {
	errMock := errors.New("mock error")
	errc := make(chan error, 1)
	go func() {
		for err := range errc {
			if err != errMock {
				t.Fail()
			}
		}
	}()
	jq, err := New(10, 20, errc)
	if err != nil {
		t.Fail()
	}
	jq.Add(func() error {
		return errMock
	})
	time.Sleep(time.Second)
	jq.Stop()
}
