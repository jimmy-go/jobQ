#!/bin/sh
# Worker pool test for database access control.
cd $GOPATH/src/github.com/jimmy-go/jobq/example/mongo
go clean
go build && \
./mongo -tasks=2000 -max-workers=3 -max-queue=10 -host=localhost -database=pompitos -username=9 -password=b
