#!/bin/sh
cd $GOPATH/src/github.com/jimmy-go/jobq
$GOBIN/gotestcover -coverprofile="cover.out" -race -covermode="count"

#go test -cover -coverprofile=coverage.out
#
#if [ "$1" == "html" ]; then
#    go tool cover -html=coverage.out
#fi
