.DEFAULT_GOAL := build
.PHONY: all fmt vet build clean

fmt:
	go fmt ./...

vet:
	go vet ./...

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build
