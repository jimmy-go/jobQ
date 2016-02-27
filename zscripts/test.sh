#!/bin/sh
cd $GOPATH/src/github.com/jimmy-go/jobq

if [ "$1" == "normal" ]; then
    go test -cover -coverprofile=coverage.out
    go tool cover -html=coverage.out
fi

if [ "$1" == "x" ]; then
    rm $GOBIN/gotestcover
    go get -u github.com/pierrre/gotestcover
    go get -u github.com/mattn/goveralls
    $GOBIN/gotestcover -coverprofile="cover.out" -race -covermode="count"
    $GOBIN/goveralls -coverprofile="cover.out"
fi

if [ "$1" == "html" ]; then
    go tool cover -html=coverage.out
fi
