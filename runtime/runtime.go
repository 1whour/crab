package runtime

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/antlabs/cronex"
	"github.com/antlabs/gstl/cmp"
	"github.com/antlabs/gstl/rwmap"
	"github.com/gnh123/ktuo/executer"
	"github.com/gnh123/ktuo/gatesock"
	"github.com/gnh123/ktuo/model"
	"github.com/gnh123/ktuo/slog"
	"github.com/gnh123/ktuo/utils"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	defautlClient   *clientv3.Client
	intervalTime    = 300 * time.Millisecond
	maxIntervalTime = 2 * time.Second
)

// 负责连接到gate服务
// 1. 如果是内网模式，runtime和gate在互相可访达的网络, 直接从etcd watch gate节点信息
// 2. 如果是外网模式，runtime只要写一个或者多个Endpoint, 做客户的负载均衡
// 3. 可以执行http, shell, grpc任务
type Runtime struct {
	cron         *cronex.Cronex
	EtcdAddr     []string      `clop:"short;long;greedy" usage:"etcd address"`
	Endpoint     []string      `clop:"long" usage:"endpoint address"`
	Level        string        `clop:"short;long" usage:"log level" default:"error"`
	WriteTimeout time.Duration `clop:"short;long" usage:"Timeout when writing messages" default:"3s"`
	// 节点名称，如果不填写，默认是uuid
	NodeName string `clop:"short;long" usage:"node name"`
	ctx      context.Context
	*slog.Slog

	MuConn sync.Mutex //保护多个go程写同一个conn

	cronFunc rwmap.RWMap[string, cronNode]
	addrs    rwmap.RWMap[string, string]
}

type cronNode struct {
	ctx    context.Context
	cancel context.CancelFunc
	tm     cronex.TimerNoder
}

func (c *cronNode) close() {
	c.cancel()
	c.tm.Stop()
}

func (r *Runtime) Init() (err error) {

	// 如果节点名为空，默认给个uuid
	if r.NodeName == "" {
		r.NodeName = uuid.New().String()
	}

	r.cron = cronex.New()
	r.ctx = context.TODO()

	// runtime被内嵌到lambda里面，可能Slog已经被初始化过
	if r.Slog == nil {
		r.Slog = slog.New(os.Stdout).SetLevel(r.Level).Str("runtime", r.NodeName)
	}

	if len(r.EtcdAddr) == 0 && len(r.Endpoint) == 0 {
		return fmt.Errorf("etcd address is nil or endpoint is nil")
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
			r.addrs.Store(string(kv.Value), string(kv.Key))
		}
	}

	for i, a := range r.Endpoint {
		r.addrs.Store(a, fmt.Sprintf("endpoint index:%d", i))
	}

	r.cron.Start()
	return nil
}

// watch etcd里面的gate地址的变化
func (r *Runtime) watchGateNode() {
	// 直接指定的Endpoint地址，没走etcd发现逻辑
	if len(r.EtcdAddr) == 0 {
		return
	}

	rsp, err := defautlClient.Get(r.ctx, model.GateNodePrefix, clientv3.WithPrefix())
	if err != nil {
		r.Warn().Msgf("runtime.get gate node %s\n", err)
	}

	r.Debug().Msgf("runtime.watchGateNode:%v\n", rsp)
	for _, ev := range rsp.Kvs {
		r.addrs.Store(string(ev.Value), string(ev.Key))
	}

	rev := rsp.Header.Revision + 1
	readGateNode := defautlClient.Watch(r.ctx, model.GateNodePrefix, clientv3.WithPrefix(), clientv3.WithRev(rev))
	for ersp := range readGateNode {
		for _, ev := range ersp.Events {
			switch {
			case ev.IsCreate():
				// 把新的gate地址加到当前addrs里面
				r.addrs.Store(string(ev.Kv.Value), string(ev.Kv.Key))
				r.Debug().Msgf("watchGateNode:create gate value(%s), key(%s)\n", ev.Kv.Value, ev.Kv.Key)
			case ev.IsModify():
				// 更新addrs里面的状态
				r.addrs.Store(string(ev.Kv.Value), string(ev.Kv.Key))
				r.Debug().Msgf("watchGateNode:modify gate value(%s), key(%s)\n", ev.Kv.Value, ev.Kv.Key)
			case ev.Type == clientv3.EventTypeDelete:
				// 把被删除的gate从当前addrs里面移除
				r.addrs.Delete(string(ev.Kv.Value))
				r.Debug().Msgf("watchGateNode:delete gate value(%s), key(%s)\n", ev.Kv.Value, ev.Kv.Key)
			}
		}
	}

	panic("watchGateNode end")
}

// 写回错误的结果, TODO，可能通过http返回
func (r *Runtime) writeError(conn *websocket.Conn, to time.Duration, code int, msg string) (err error) {
	r.MuConn.Lock()
	err = utils.WriteJsonTimeout(conn, model.RuntimeResp{Code: code, Message: msg}, to)
	r.MuConn.Unlock()
	return err
}

func (r *Runtime) removeFromExec(param *model.Param) ([]byte, error) {
	e, ok := r.cronFunc.LoadAndDelete(param.Executer.TaskName)
	if !ok {
		return nil, fmt.Errorf("not found taskName:%s", param.Executer.TaskName)
	}
	e.close()
	r.Debug().Msgf("action(%s), task is remove:%s, tm:%p\n", param.Action, param.Executer.TaskName, e.tm)
	return nil, nil
}

func (r *Runtime) createToExec(ctx context.Context, param *model.Param) ([]byte, error) {
	e, err := executer.CreateExecuter(ctx, param)
	if err != nil {
		r.Error().Msgf("param.TaskName(%s) create fail:%s\n", param.Executer.TaskName, err)
		return nil, err
	}

	return e.Run()
}

func (r *Runtime) createCron(param *model.Param) (b []byte, err error) {

	ctx, cancel := context.WithCancel(r.ctx)
	tm, err := r.cron.AddFunc(param.Trigger.Cron, func() {
		// 创建执行器
		payload, err := r.createToExec(ctx, param)
		if err != nil {
			r.Error().Msgf("createToExec %s, taskName:%s\n", err, param.Executer.TaskName)
			return
		}
		// TODO: 错误要上报到gate模块
		// TODO: payload
		r.Debug().Msgf("result:%s", payload)
	})

	if err != nil {
		cancel()
		tm.Stop()
		return nil, err
	}

	old, ok := r.cronFunc.Load(param.Executer.TaskName)
	if ok {
		old.close()
	}
	// 按道理不应该old有值
	r.Debug().Msgf("old(%t), createCron tm:%p, taskName:%s\n", ok, tm, param.Executer.TaskName)
	r.cronFunc.Store(param.Executer.TaskName, cronNode{ctx: ctx, cancel: cancel, tm: tm})
	return nil, nil
}

func (r *Runtime) runCrudCmd(conn *websocket.Conn, param *model.Param) (payload []byte, err error) {

	switch {
	case param.IsCreate():
		r.createCron(param)

	case param.IsRemove(), param.IsStop():
		// 删除和stop对于runtime是一样，停止当前运行的，然后从sync.Map删除
		payload, err = r.removeFromExec(param)
	case param.IsUpdate():
		// 先删除
		if payload, err = r.removeFromExec(param); err != nil {
			return payload, err
		}
		payload, err = r.createCron(param)
	}
	return payload, err
}

type interval time.Duration

func (i *interval) reset() {
	*i = interval(intervalTime)
}

func (i *interval) sleep() {

	time.Sleep(time.Duration(*i))
	*i *= 2
	*i = interval(cmp.Min(time.Duration(*i), maxIntervalTime))
}

// 初始化时创建 只创建一个长连接
// 故意这么设计
// 为了简化gate广播发送的逻辑, 一个runtime只会连一个gate，并且只有一个长连接，
// 这样不需要考虑去重，引入额外中间件，简化设计
func (r *Runtime) createConnRand(lambda bool) (err error) {

	var t interval
	t.reset()

	id := uuid.New().String()
	for {

		addrs := r.addrs.Keys()
		if len(addrs) == 0 {
			r.Info().Msgf("no gate address available\n")
			t.sleep()
			continue
		}

		addr := utils.SliceRandOne(addrs)

		for i := 0; i < 2; i++ {
			r.Debug().Msgf("# addr is %s, id:%s", addr, id)
			gs := gatesock.New(r.Slog, r.runCrudCmd, addr, r.NodeName, r.WriteTimeout, &r.MuConn, lambda, id)
			if err := gs.CreateConntion(); err != nil {
				// 如果握手或者上传第一个包失败，sleep 下，再重连一次
				r.Error().Msgf("createConnection fail:%v\n", err)
				t.sleep()
			} else {
				t.reset()
			}
		}
	}
}

func (r *Runtime) Run(lambda bool, initAfter func()) error {

	if err := r.Init(); err != nil {
		return err
	}

	if initAfter != nil {
		initAfter()
	}
	if len(r.EtcdAddr) > 0 {
		go r.createConnRand(lambda)
		r.watchGateNode()
		return nil
	}

	return r.createConnRand(lambda)
}

// 该模块的入口函数
// 命令行里面的子命令会调用
func (r *Runtime) SubMain() {
	if err := r.Run(false, nil); err != nil {
		os.Exit(1)
	}
}
