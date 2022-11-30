# Use goreman to run `go get github.com/mattn/goreman`
# Change the path of bin/etcd if etcd is located elsewhere

# mock服务，用于检查http执行的次数可对
ktuo.mocksrv: ./ktuo mocksrv
#scheduler.gate-auto1:    ./scheduler gate -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -a -l debug
#scheduler.gate-atuo2:    ./scheduler gate -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -a -l debug
ktuo.gate1:    ./ktuo gate -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -l debug -s "127.0.0.1:3434"
ktuo.gate2:    ./ktuo gate -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -l debug -s "127.0.0.1:3535"
ktuo.runtime1: ./ktuo runtime -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 --level debug
ktuo.runtime2: ./ktuo runtime -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 --level debug
ktuo.mjobs1:   ./ktuo mjobs -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -l debug
ktuo.mjobs2:   ./ktuo mjobs -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -l debug

