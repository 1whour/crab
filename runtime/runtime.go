package runtime

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/antlabs/cronex"
	"github.com/antlabs/gstl/cmp"
	"github.com/antlabs/gstl/rwmap"
	"github.com/gnh123/scheduler/executer"
	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/slog"
	"github.com/gnh123/scheduler/utils"
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
// 2. 如果是外网模式，runtime只要写一个或者多个GateAddr, 做客户的负载均衡
// 3. 可以执行http, shell, grpc任务
type Runtime struct {
	cron *cronex.Cronex
	//cron         *cron.Cron
	EtcdAddr     []string      `clop:"short;long;greedy" usage:"etcd address"`
	GateAddr     []string      `clop:"long" usage:"endpoint address"`
	Level        string        `clop:"short;long" usage:"log level" default:"error"`
	WriteTimeout time.Duration `clop:"short;long" usage:"Timeout when writing messages" default:"3s"`
	Name         string        `clop:"short;long" usage:"node name"`
	ctx          context.Context
	*slog.Slog

	muWc sync.Mutex //保护多个go程写同一个conn

	cronFunc rwmap.RWMap[string, cronNode]
	addrs    rwmap.RWMap[string, string]
}

type cronNode struct {
	ctx    context.Context
	cancel context.CancelFunc
	//tm     cron.EntryID
	tm cronex.TimerNoder
}

func (c *cronNode) close() {
	c.cancel()
	c.tm.Stop()
}

func (r *Runtime) init() (err error) {

	// 如果节点名为空，默认给个uuid
	if r.Name == "" {
		r.Name = uuid.New().String()
	}

	//r.cron = cron.New()
	//r.cron = cron.New(cron.WithParser(cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)))
	r.cron = cronex.New()
	r.ctx = context.TODO()
	r.Slog = slog.New(os.Stdout).SetLevel(r.Level).Str("runtime", r.Name)

	if len(r.EtcdAddr) == 0 && len(r.GateAddr) == 0 {
		r.Error().Msg("etcd address is nil or endpoint is nil")
		os.Exit(1)
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

	for i, a := range r.GateAddr {
		r.addrs.Store(a, fmt.Sprintf("endpoint index:%d", i))
	}

	r.cron.Start()
	return nil
}

// watch etcd里面的gate地址的变化
func (r *Runtime) watchGateNode() {
	// 直接指定的GateAddr地址，没走etcd发现逻辑
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

func (r *Runtime) writeWhoami(conn *websocket.Conn) (err error) {
	r.muWc.Lock()
	err = utils.WriteJsonTimeout(conn, model.Whoami{Name: r.Name}, r.WriteTimeout)
	r.muWc.Unlock()
	return err
}

// 写回错误的结果, TODO，可能通过http返回
func (r *Runtime) writeError(conn *websocket.Conn, to time.Duration, code int, msg string) (err error) {
	r.muWc.Lock()
	err = utils.WriteJsonTimeout(conn, model.RuntimeResp{Code: code, Message: msg}, to)
	r.muWc.Unlock()
	return err
}

func (r *Runtime) removeFromExec(param *model.Param) error {
	e, ok := r.cronFunc.LoadAndDelete(param.Executer.TaskName)
	if !ok {
		return fmt.Errorf("not found taskName:%s", param.Executer.TaskName)
	}
	e.close()
	r.Debug().Msgf("action(%s), task is remove:%s, tm:%p\n", param.Action, param.Executer.TaskName, e.tm)
	return nil
}

func (r *Runtime) createToExec(ctx context.Context, param *model.Param) error {
	e, err := executer.CreateExecuter(ctx, param)
	if err != nil {
		r.Error().Msgf("param.TaskName(%s) create fail:%s\n", param.Executer.TaskName, err)
		return err
	}

	if err := e.Run(); err != nil {
		return err
	}
	return nil
}

func (r *Runtime) createCron(param *model.Param) (err error) {

	ctx, cancel := context.WithCancel(r.ctx)
	tm, err := r.cron.AddFunc(param.Trigger.Cron, func() {
		// 创建执行器
		var err error
		err = r.createToExec(ctx, param)
		if err != nil {
			r.Error().Msgf("createToExec %s, taskName:%s\n", err, param.Executer.TaskName)
		}
		// TODO 错误要上报到gate模块
	})

	if err != nil {
		cancel()
		tm.Stop()
		return err
	}

	old, ok := r.cronFunc.Load(param.Executer.TaskName)
	if ok {
		old.close()
	}
	// 按道理不应该old有值
	r.Debug().Msgf("old(%t), createCron tm:%p, taskName:%s\n", ok, tm, param.Executer.TaskName)
	r.cronFunc.Store(param.Executer.TaskName, cronNode{ctx: ctx, cancel: cancel, tm: tm})
	return nil
}

func (r *Runtime) runCrudCmd(conn *websocket.Conn, param *model.Param) (err error) {

	switch {
	case param.IsCreate():
		r.createCron(param)

	case param.IsRemove(), param.IsStop():
		// 删除和stop对于runtime是一样，停止当前运行的，然后从sync.Map删除
		err = r.removeFromExec(param)
	case param.IsUpdate():
		// 先删除
		if err = r.removeFromExec(param); err != nil {
			return err
		}
		err = r.createCron(param)
	}
	return err
}

// 接受来自gate服务的命令, 执行并返回结果
func (r *Runtime) readLoop(conn *websocket.Conn) error {

	r.Debug().Msgf("call readLoop\n")
	go func() {
		// 对conn执行心跳检查，conn可能长时间空闲，为是检查conn是否健康，加上心跳
		for {
			time.Sleep(model.RuntimeKeepalive)
			if err := r.writeWhoami(conn); err != nil {
				r.Warn().Msgf("write whoami:%s\n", err)
				conn.Close() //关闭conn. ReadJOSN也会出错返回
				return
			}
		}
	}()

	for {
		var param model.Param
		err := conn.ReadJSON(&param) //这里不加超时时间, 一直监听gate推过来的信息
		if err != nil {
			return err
		}

		go func() {
			r.Debug().Msgf("crud action:%s, taskName:%s\n", param.Action, param.Executer.TaskName)
			if err := r.runCrudCmd(conn, &param); err != nil {
				r.Error().Msgf("runtime.runCrud, action(%s):%s\n", param.Action, err)
				//r.writeError(conn, r.WriteTimeout, 1, err.Error())
				return
			}
		}()
	}

}

func genGateAddr(gateAddr string) string {
	if strings.HasPrefix(gateAddr, "ws://") || strings.HasPrefix(gateAddr, "wss://") {
		return gateAddr
	}
	return "ws://" + gateAddr
}

// 创建一个长连接
func (r *Runtime) createConntion(gateAddr string) error {

	gateAddr = genGateAddr(gateAddr) + model.TASK_STREAM_URL
	c, _, err := websocket.DefaultDialer.Dial(gateAddr, nil)
	if err != nil {
		r.Error().Msgf("runtime:dial:%s, address:%s\n", err, gateAddr)
		return err
	}

	defer c.Close()
	err = utils.WriteJsonTimeout(c, model.Whoami{Name: r.Name}, r.WriteTimeout)
	if err != nil {
		return err
	}

	return r.readLoop(c)
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
func (r *Runtime) createConnRand() {

	var t interval
	t.reset()

	for {

		addrs := r.addrs.Keys()
		if len(addrs) == 0 {
			r.Info().Msgf("no gate address available\n")
			t.sleep()
			continue
		}

		addr := utils.SliceRandOne(addrs)

		for i := 0; i < 2; i++ {

			if err := r.createConntion(addr); err != nil {
				// 如果握手或者上传第一个包失败，sleep 下，再重连一次
				r.Error().Msgf("createConnection fail:%v\n", err)
				t.sleep()
			} else {
				t.reset()
			}
		}
	}
}

// 该模块的入口函数
func (r *Runtime) SubMain() {

	r.init()
	go r.createConnRand()
	r.watchGateNode()
}
