#!/bin/bash


go build -tags "h264enc" cmd/agent.go


git push -u origin main

GOOS=linux GOARCH=arm64 go build -tags "vp8enc" cmd/agent.go
