#!/bin/sh
cd $GOPATH/src/github.com/jimmy-go/jobq
go test -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
