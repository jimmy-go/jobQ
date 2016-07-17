#!/bin/sh
cd $GOPATH/src/github.com/jimmy-go/jobq

if [ "$1" == "bench" ]; then
    go test -race -bench=.
    exit;
fi

if [ "$1" == "x" ]; then
    go get -u github.com/pierrre/gotestcover
    go get -u github.com/mattn/goveralls
    $GOBIN/gotestcover -coverprofile="cover.out" -race -covermode="count"
    $GOBIN/goveralls -coverprofile="cover.out"
    exit;
fi

if [ "$1" == "html" ]; then
    go test -cover -coverprofile=coverage.out
    go tool cover -html=coverage.out
    exit;
fi

go test -cover -coverprofile=coverage.out
