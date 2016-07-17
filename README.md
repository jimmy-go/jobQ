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

Benchmark:
```
Benchmark100x100-4  	   10000	    107528 ns/op
Benchmark10x100-4   	   10000	   1081856 ns/op
Benchmark1x1-4      	     100	  10495713 ns/op
Benchmark1x2-4      	     100	  10485961 ns/op
Benchmark1x3-4      	     100	  10410480 ns/op
Benchmark1000x1000-4	  200000	     11283 ns/op
Benchmark10x30-4    	    2000	   1063811 ns/op
Benchmark1x100-4    	   10000	  10748689 ns/op
```

License:

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
