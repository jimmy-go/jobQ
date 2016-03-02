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
jq, err := jobq.New(*ws, *qlen)
```

Add jobs:

```go
task := func() error {
    // do stuff...
}
// Send the job to the queue.
jq.Add(task)
```

Stop the pool:

```go
jq.Stop()

```

###### Example:

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
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	log.Printf("workers [%d]", *ws)
	log.Printf("queue len [%d]", *qlen)
	log.Printf("tasks [%d]", *tasks)
	w := 650 * time.Millisecond
	log.Printf("time for mock task [%v]", w)

	done := make(chan struct{}, 1)
	// want to see how many goroutines are running.
	go func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				log.Printf("main : GOROUTINES [%v]", runtime.NumGoroutine())
			case <-done:
				log.Printf("done!")
				return
			}
		}
	}()

	// declare a new worker pool

	// ws: workers size count.
	// qlen: size for queue length. All left jobs will wait until queue release some slot.
	jq, err := jobq.New(*ws, *qlen)
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
	done <- struct{}{}
	time.Sleep(15 * time.Second)
	panic(errors.New("see goroutines"))
}
```

Benchmark:

```
Benchmark100x100-4  	   10000	    122320 ns/op
Benchmark10x100-4   	   10000	   1131420 ns/op
Benchmark1x1-4      	     100	  11379068 ns/op
Benchmark1x2-4      	     100	  11023856 ns/op
Benchmark1x3-4      	     100	  10712324 ns/op
Benchmark1000x1000-4	   50000	     35176 ns/op
Benchmark10x30-4    	    2000	   1145672 ns/op
Benchmark1x100-4    	   10000	  11455667 ns/op
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

[![Bitdeli Badge](https://d2weczhvl823v0.cloudfront.net/jimmy-go/jobq/trend.png)](https://bitdeli.com/free "Bitdeli Badge")
