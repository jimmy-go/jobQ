#!/bin/sh
# Worker pool test for database access control.
cd $GOPATH/src/github.com/jimmy-go/jobq/examples/mongo
go build -o $GOBIN/mongojobqs && \
$GOBIN/mongojobqs -tasks=2000 -max-workers=3 -max-queue=10 -host=localhost -database=pompitos -username=8Y59e0DUcf -password=yi7Sry1KEb
