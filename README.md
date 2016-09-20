####Job queue in go with support for custom workers.

[![License MIT](https://img.shields.io/npm/l/express.svg)](http://opensource.org/licenses/MIT)
[![Build Status](https://travis-ci.org/jimmy-go/jobQ.svg?branch=master)](https://travis-ci.org/jimmy-go/jobQ)
[![Go Report Card](https://goreportcard.com/badge/github.com/jimmy-go/jobq?1)](https://goreportcard.com/report/github.com/jimmy-go/jobq)
[![GoDoc](http://godoc.org/github.com/jimmy-go/jobq?status.png)](http://godoc.org/github.com/jimmy-go/jobq)
[![Coverage Status](https://coveralls.io/repos/github/jimmy-go/jobQ/badge.svg?branch=master&1)](https://coveralls.io/github/jimmy-go/jobQ?branch=master)

######Install:
```
go get gopkg.in/jimmy-go/jobq.v2
```

######Usage:

Declare a new worker pool:

```go
// ws: workers size count.
// qlen: size for queue length. All left jobs will wait until queue release some slot.
// timeout: timeout for tasks.
jq, err := jobq.New(ws, qlen, time.Duration(time.Second))

// Add tasks
task := func(cancel chan struct{}) error {
    // do stuff...
}
jq.AddTask(task)

// Stop the pool
jq.Stop()

// Wait blocks until Stop is called and tasks are completed.
jq.Wait()
```

######Benchmark:
```
```

######Old version:
`go get gopkg.in/jimmy-go/jobq.v1`

######License:

The MIT License (MIT)

Copyright (c) 2016 Angel Del Castillo

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
