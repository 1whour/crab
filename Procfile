# Use goreman to run `go get github.com/mattn/goreman`
# Change the path of bin/etcd if etcd is located elsewhere

scheduler.gate1:    ./scheduler gate -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -a -l debug
scheduler.gate2:    ./scheduler gate -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -a -l debug
scheduler.runtime1: ./scheduler runtime -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 --level debug
scheduler.runtime2: ./scheduler runtime -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 --level debug
scheduler.mjobs1:   ./scheduler mjobs -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -l debug
scheduler.mjobs2:   ./scheduler mjobs -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -l debug

