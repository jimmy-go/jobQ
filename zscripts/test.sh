#!/bin/sh
cd $GOPATH/src/github.com/jimmy-go/jobq

if [ "$1" == "bench" ]; then
    go test -race -bench=.
    exit;
fi

if [ "$1" == "html" ]; then
    go test -cover -coverprofile=coverage.out
    go tool cover -html=coverage.out
    exit;
fi

if [ "$1" == "full" ]; then
    go test -test.run=$2
    exit;
fi

go test -race -cover -coverprofile=coverage.out -test.run=$1
