#!/bin/sh
cd $GOPATH/src/github.com/jimmy-go/jobq/examples/race

go build -race -o $GOBIN/wprace && $GOBIN/wprace
