all: build

cluster:
	- killall crab
	- rm cluster.log
	goreman start|tee -a cluster.log

norace:
	go build ./cmd/crab/crab.go

build:
	go build -race ./cmd/crab/crab.go

test.shell:
	bash ./testscript/run.sh
test:
	go test ./...
	bash ./testscript/run.sh
