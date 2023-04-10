#!/bin/bash
#

find_bugs_golint() {
	basedir=$(pwd)
	cd $basedir
	for element in `find . -path ./.cache -prune -o -name "*.go"`; do
		gofmt -w ${basedir}/$element
	done
}

find_bugs_golint $@

go build -tags "h264enc" cmd/agent.go
