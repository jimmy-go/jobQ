#!/bin/sh
cd $GOPATH/src/github.com/jimmy-go/jobq/examples/race

rm $GOBIN/wprace

#go build -race -gcflags -m -o $GOBIN/wprace && $GOBIN/wprace
go build -race -o $GOBIN/wprace && $GOBIN/wprace
