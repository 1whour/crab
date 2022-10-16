package runtime

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/slog"
	"github.com/gnh123/scheduler/utils"
	"github.com/gorilla/websocket"
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
	Level    string   `clop:"long" usage:"log level" default:"error"`

	ctx context.Context
	*slog.Slog
	addr sync.Map
}

func (r *Runtime) init() (err error) {
	r.Slog = slog.New(os.Stdout).SetLevel(r.Level)

	if len(r.EtcdAddr) == 0 && len(r.Endpoint) == 0 {
		panic("etcd address is nil or endpoint is nil")
	}
	// 设置日志
	if len(r.EtcdAddr) > 0 {
		if defautlClient, err = utils.NewEtcdClient(r.EtcdAddr); err != nil {
			return err
		}

		rsp, err := defautlClient.Get(r.ctx, model.GateNodePrefix, clientv3.WithPrefix())
		if err != nil {
			return err
		}

		for _, kv := range rsp.Kvs {
			r.addr.Store(kv.Value, kv.Key)
		}
	}

	for i, a := range r.Endpoint {
		r.addr.Store(a, fmt.Sprintf("endpoint index:%d", i))
	}

	return nil
}

// watch etcd里面的gate地址的变化
func (r *Runtime) watchGateNode() {

	readGateNode := defautlClient.Watch(r.ctx, model.GateNodePrefix, clientv3.WithPrefix())
	for ersp := range readGateNode {
		for _, ev := range ersp.Events {
			switch {
			case ev.IsCreate():
				// 创建新的read/write loop
			case ev.IsModify():
			case ev.Type == clientv3.EventTypeDelete:
			}
		}
	}
}

// 接受来自gate服务的命令, 执行并返回结果
func (r *Runtime) readLoop(conn *websocket.Conn) error {
	for {
		err := conn.ReadJSON()
		if err != nil {
			return err
		}
		// TODO 运行执行器
	}

}

func (r *Runtime) createConntion(gateAddr string, val string) {

	c, _, err := websocket.DefaultDialer.Dial(gateAddr, nil)
	if err != nil {
		r.Error().Msgf("runtime:dial:%s\n", err)
		return
	}

	defer c.Close()
	r.readLoop(c)
}

// 初始化时创建
func (r *Runtime) createMultipleConn() {
	r.addr.Range(func(key, val any) bool {
		go r.createConntion(key.(string), val.(string))
		return true
	})
}

// 该模块的入口函数
func (r *Runtime) SubMain() {

	r.init()
	r.createMultipleConn()
	r.watchGateNode()
}
