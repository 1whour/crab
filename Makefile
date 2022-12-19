all: build

cluster:
	- killall crab
	- rm cluster.log
	goreman -f cluster.proc start|tee -a cluster.log

monomer:
	echo "monomer"
	- killall crab
	- rm monomer.log
	goreman -f monomer.proc start|tee -a monomer.log

norace:
	go build ./cmd/crab/crab.go

build:
	go build -race ./cmd/crab/crab.go

test.shell:
	bash ./testscript/run.sh
test:
	go test ./...
	bash ./testscript/run.sh
