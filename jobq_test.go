// Package jobq contains type JobQ for worker pools.
package jobq

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newWorkers(l int) []Worker {
	var ws []Worker
	for i := 0; i < l; i++ {
		w, _ := NewMyWorker(time.Second)
		ws = append(ws, w)
	}
	return ws
}

func TestNew(t *testing.T) {
	ws := newWorkers(1)

	table := []struct {
		Purpose string
		Workers []Worker
		Size    int
		Err     error
	}{
		{"1. OK", ws, 10, nil},
	}
	for _, x := range table {
		_, err := New(x.Workers, x.Size)
		assert.Equal(t, x.Err, err, x.Purpose)
	}
}

func TestRun(t *testing.T) {
	ws := newWorkers(3)

	p, err := New(ws, 5)
	assert.Nil(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	task := func() (context.Context, interface{}) {
		ctx := context.TODO()

		vars := url.Values{}
		vars.Set("a", "1")
		return ctx, &Payload{ts.URL, vars}
	}
	err = p.AddTask(task)
	assert.Nil(t, err)
}
