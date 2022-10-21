all: build

build:
	go build ./cmd/scheduler/scheduler.go

build-race:
	go build -race ./cmd/scheduler/scheduler.go

test:
	bash ./testscript/run.sh

test.create:
	bash ./testscript/run.sh create

test.delete:
	bash ./testscript/run.sh delete
