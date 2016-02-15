package jobq

import (
	"errors"
	"testing"
	"time"
)

var (
	errMock = errors.New("mock error")
)

// TODO; add some tests and benchmark.

type T struct {
	Size     int
	Len      int
	Expected error
}

var tests = []T{
	T{1, 2, nil},
	T{7, 6, nil},
	T{1, -1, errInvalidQueueSize},
	T{-1, 1, errInvalidWorkerSize},
}

func TestNew(t *testing.T) {
	for _, m := range tests {
		errc := make(chan error, 1)
		_, err := New(m.Size, m.Len, errc)
		if err != m.Expected {
			t.Fail()
		}
	}
}

func TestWork(t *testing.T) {
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
