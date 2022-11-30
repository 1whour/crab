all: build

cluster:
	- killall ktuo
	- rm cluster.log
	goreman start|tee -a cluster.log

norace:
	go build ./cmd/ktuo/ktuo.go

build:
	go build -race ./cmd/ktuo/ktuo.go

test.shell:
	bash ./testscript/run.sh
test:
	go test ./...
	bash ./testscript/run.sh
