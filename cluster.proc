# Use goreman to run `go get github.com/mattn/goreman`
# Change the path of bin/etcd if etcd is located elsewhere

# mock服务，用于检查http执行的次数可对
crab.mocksrv: ./crab mocksrv
#scheduler.gate-auto1:    ./scheduler gate -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -a -l debug
#scheduler.gate-atuo2:    ./scheduler gate -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -a -l debug
crab.gate1:    ./crab gate -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -l debug -s "127.0.0.1:3434" --dsn "root:3434@@Aa@tcp(127.0.0.1:3306)/crab?charset=utf8mb4&parseTime=True&loc=Local"
crab.gate2:    ./crab gate -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -l debug -s "127.0.0.1:3535" --dsn "root:3434@@Aa@tcp(127.0.0.1:3306)/crab?charset=utf8mb4&parseTime=True&loc=Local"
crab.runtime1: ./crab runtime -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 --level debug
crab.runtime2: ./crab runtime -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 --level debug
crab.mjobs1:   ./crab mjobs -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -l debug
crab.mjobs2:   ./crab mjobs -e 127.0.0.1:32379 127.0.0.1:22379 127.0.0.1:2379 -l debug

