#### Job Queue in Go

Simple but powerful job queue in go.

[![License MIT](https://img.shields.io/npm/l/express.svg)](http://opensource.org/licenses/MIT)
[![Build Status](https://travis-ci.org/jimmy-go/jobQ.svg?branch=master)](https://travis-ci.org/jimmy-go/jobQ)
[![Go Report Card](https://goreportcard.com/badge/github.com/jimmy-go/jobq?1)](https://goreportcard.com/report/github.com/jimmy-go/jobq)
[![GoDoc](http://godoc.org/github.com/jimmy-go/jobq?status.png)](http://godoc.org/github.com/jimmy-go/jobq)
[![Coverage Status](https://coveralls.io/repos/github/jimmy-go/jobQ/badge.svg?branch=master&1)](https://coveralls.io/github/jimmy-go/jobQ?branch=master)

----

##### Usage:

Declare a new worker pool:

```go
// ws: workers size count.
// qlen: size for queue length. All left jobs will wait until queue release some slot.
// errc: channel for errors in execution.
jq, err := jobq.New(*ws, *qlen, errc)
```

Add jobs:

```go
task := func() error {
    // do stuff...
}
// send the job to the queue.
jq.Add(task)

...
// stop the pool
jq.Stop()
```

Example:

```go
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
	ws    = flag.Int("workers", 5, "Number of workers.")
	qlen  = flag.Int("queue", 10, "Number of queue works.")
	tasks = flag.Int("tasks", 100, "Number of tasks.")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	log.SetFlags(log.Lshortfile)
	log.Printf("workers [%d]", *ws)
	log.Printf("queue len [%d]", *qlen)
	log.Printf("tasks [%d]", *tasks)
	w := 650 * time.Millisecond
	log.Printf("time for mock task [%v]", w)

	// want to see how many goroutines are running.
	go func() {
		for {
			log.Printf("main : GOROUTINES [%v]", runtime.NumGoroutine())
			time.Sleep(5 * time.Second)
		}
	}()

	// view errors
	errc := make(chan error)
	go func() {
		for err := range errc {
			if err != nil {
				log.Printf("main : error channel : err [%s]", err)
			}
		}
	}()

	// declare a new worker pool

	// ws: workers size count.
	// qlen: size for queue length. All left jobs will wait until queue release some slot.
	// errc: channel for errors in execution.
	jq, err := jobq.New(*ws, *qlen, errc)
	if err != nil {
		log.Printf("main : err [%s]", err)
	}

	for i := 0; i < *tasks; i++ {
		func(index int) {
			// task satisfies type jobq.Job, can be any function with error return.
			// I take this aproximation because some worker pool I see around use an interface
			// what I consider a limiting factor.
			// this way you only need declare a function and you're ready to go!
			// [if you think I'm doing it wrong please tell me :)]
			task := func() error {
				time.Sleep(w)
				log.Printf("main : task [%d] done!", index)
				return nil
			}
			// send the job to the queue.
			jq.Add(task)
		}(i)
	}

	log.Println("sleep 15 second!")
	time.Sleep(15 * time.Second)
	jq.Stop()
	time.Sleep(15 * time.Second)
	panic(errors.New("see goroutines"))
}
```

LICENSE:

>The MIT License (MIT)
>
>Copyright (c) 2016 Angel Del Castillo
>
>Permission is hereby granted, free of charge, to any person obtaining a copy
>of this software and associated documentation files (the "Software"), to deal
>in the Software without restriction, including without limitation the rights
>to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
>copies of the Software, and to permit persons to whom the Software is
>furnished to do so, subject to the following conditions:
>
>The above copyright notice and this permission notice shall be included in all
>copies or substantial portions of the Software.
>
>THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
>IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
>FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
>AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
>LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
>OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
>SOFTWARE.
