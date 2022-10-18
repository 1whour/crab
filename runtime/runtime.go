package runtime

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"sync"
	"time"

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
// 2. 如果是外网模式，runtime只要写一个或者多个GateAddr, 做客户的负载均衡
type Runtime struct {
	EtcdAddr []string `clop:"short;long" usage:"etcd address"`
	GateAddr []string `clop:"long" usage:"endpoint address"`
	Level    string   `clop:"long" usage:"log level" default:"error"`

	ctx context.Context
	*slog.Slog

	sync.RWMutex
	addr     map[string]string
	currNode string
}

func (r *Runtime) init() (err error) {
	rand.Seed(time.Now().UnixNano())

	r.Slog = slog.New(os.Stdout).SetLevel(r.Level)

	if len(r.EtcdAddr) == 0 && len(r.GateAddr) == 0 {
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
			// 从etcd里面获取gate ip
			r.addr[string(kv.Value)] = string(kv.Key)
		}
	}

	for i, a := range r.GateAddr {
		r.addr[a] = fmt.Sprintf("endpoint index:%d", i)
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
		/*
			err := conn.ReadJSON()
			if err != nil {
				return err
			}
		*/
		// TODO 运行执行器
	}

}

func (r *Runtime) createConntion(gateAddr string) {

	c, _, err := websocket.DefaultDialer.Dial(gateAddr+"/"+model.TASK_STREAM_URL, nil)
	if err != nil {
		r.Error().Msgf("runtime:dial:%s\n", err)
		return
	}

	defer c.Close()
	r.readLoop(c)
}

// 初始化时创建
// 只创建一个长连接，故意这么设计
// 为是简化gate广播发送的逻辑, 一个runtime只会连一个gate，并且只有一个连接，不需要考虑去重
func (r *Runtime) createConnRand() {
	addrs := make([]string, 0, len(r.addr))
	//go r.createConntion(key.(string), val.(string))
	for _, ip := range addrs {
		addrs = append(addrs, ip)
	}

	sort.Strings(addrs)
	index := rand.Int31n(int32(len(addrs)))
	go r.createConntion(addrs[index])
}

// 该模块的入口函数
func (r *Runtime) SubMain() {

	r.init()
	r.createConnRand()
	r.watchGateNode()
}
