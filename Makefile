all: build

cluster:
	- killall scheduler
	- rm cluster.log
	goreman start|tee -a cluster.log

build:
	go build ./cmd/scheduler/scheduler.go

build-race:
	go build -race ./cmd/scheduler/scheduler.go

test.shell:
	bash ./testscript/run.sh
test:
	go test ./...
	bash ./testscript/run.sh
