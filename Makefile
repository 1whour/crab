all: build

build:
	go build ./cmd/scheduler/scheduler.go

build-race:
	go build -race ./cmd/scheduler/scheduler.go
