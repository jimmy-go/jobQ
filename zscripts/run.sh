#!/bin/sh
# Worker pool test for database access control.
cd $GOPATH/src/github.com/jimmy-go/jobq/example
go clean
go build && ./example
