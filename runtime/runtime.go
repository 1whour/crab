package runtime

import (
	"context"
	"sync"

	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	defautlClient *clientv3.Client
)

// 负责连接到gate服务
// 1. 如果是内网模式，runtime和gate在互相可访达的网络, 直接从etcd watch gate节点信息
// 2. 如果是外网模式，runtime只要写一个或者多个Endpoint, 做客户的负载均衡
type Runtime struct {
	EtcdAddr []string `clop:"short;long" usage:"etcd address"`
	Endpoint []string `clop:"long" usage:"endpoint address"`

	sync.RWMutex
	ctx context.Context
}

func (r *Runtime) init() (err error) {

	if len(r.EtcdAddr) > 0 {
		if defautlClient, err = utils.NewEtcdClient(r.EtcdAddr); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runtime) watchGateNode() {

	readGateNode := defautlClient.Watch(r.ctx, model.GateNodePrefix, clientv3.WithPrefix())
	for ersp := range readGateNode {
		for _, ev := range ersp.Events {
			switch {
			case ev.IsCreate():
			case ev.IsModify():
			case ev.Type == clientv3.EventTypeDelete:
			}
		}
	}
}

// 该模块的入口函数
func SubMain() {

}
