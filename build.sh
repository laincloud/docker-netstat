#! /bin/bash

docker run --rm -v $GOPATH:/go -e GOBIN=/go/src/github.com/laincloud/docker-netstat/bin golang:1.9.2 go install github.com/laincloud/docker-netstat