#!/bin/sh
# Worker pool test for database access control.
cd $GOPATH/src/github.com/jimmy-go/jobq/examples
go build -race -o $GOBIN/jobqs && $GOBIN/jobqs
