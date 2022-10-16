package utils

import (
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// 创建etcd的连接池
func NewEtcdClient(endpoints []string) (*clientv3.Client, error) {

	return clientv3.New(clientv3.Config{
		//Endpoints:   []string{"localhost:2379", "localhost:22379", "localhost:32379"},
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})

}
