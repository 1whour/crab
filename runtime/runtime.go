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
	"github.com/antlabs/gstl/mapex"
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
	cron         *cronex.Cronex
	EtcdAddr     []string      `clop:"short;long;greedy" usage:"etcd address"`
	GateAddr     []string      `clop:"long" usage:"endpoint address"`
	Level        string        `clop:"short;long" usage:"log level" default:"error"`
	WriteTimeout time.Duration `clop:"short;long" usage:"Timeout when writing messages" default:"3s"`
	Name         string        `clop:"short;long" usage:"node name"`
	ctx          context.Context
	*slog.Slog

	muWc sync.Mutex //保护多个go程写同一个conn

	exec sync.Map
	sync.RWMutex
	addrs map[string]string
}

// TODO 在gstl里面实现下给map套个sync.RWMutex的代码
func (r *Runtime) storeAddr(key, value string) {
	r.Lock()
	r.addrs[key] = value
	r.Unlock()
}

func (r *Runtime) deleteAddr(key string) {
	r.Lock()
	delete(r.addrs, key)
	r.Unlock()
}

func (r *Runtime) keyAddr() []string {

	r.RLock()
	addrs := mapex.Keys(r.addrs)
	r.RUnlock()
	return addrs
}

func (r *Runtime) init() (err error) {

	// 如果节点名为空，默认给个uuid
	if r.Name == "" {
		r.Name = uuid.New().String()
	}

	r.addrs = make(map[string]string)
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
			r.addrs[string(kv.Value)] = string(kv.Key)
		}
	}

	for i, a := range r.GateAddr {
		r.addrs[a] = fmt.Sprintf("endpoint index:%d", i)
	}

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
		r.storeAddr(string(ev.Value), string(ev.Key))
	}

	rev := rsp.Header.Revision + 1
	readGateNode := defautlClient.Watch(r.ctx, model.GateNodePrefix, clientv3.WithPrefix(), clientv3.WithRev(rev))
	for ersp := range readGateNode {
		for _, ev := range ersp.Events {
			switch {
			case ev.IsCreate():
				// 把新的gate地址加到当前addrs里面
				r.storeAddr(string(ev.Kv.Value), string(ev.Kv.Key))
				r.Debug().Msgf("create gate value(%s), key(%s)\n", ev.Kv.Value, ev.Kv.Key)
			case ev.IsModify():
				// 更新addrs里面的状态
				r.storeAddr(string(ev.Kv.Value), string(ev.Kv.Key))
				r.Debug().Msgf("modify gate value(%s), key(%s)\n", ev.Kv.Value, ev.Kv.Key)
			case ev.Type == clientv3.EventTypeDelete:
				// 把被删除的gate从当前addrs里面移除
				r.deleteAddr(string(ev.Kv.Value))
				r.Debug().Msgf("delete gate value(%s), key(%s)\n", ev.Kv.Value, ev.Kv.Key)
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
	e, ok := r.exec.Load(param.Executer.TaskName)
	if !ok {
		return fmt.Errorf("not found taskName:%s", param.Executer.TaskName)
	}
	err := e.(executer.Executer).Stop()
	r.exec.Delete(param.Executer.TaskName)
	return err
}

func (r *Runtime) createToExec(param *model.Param) error {
	e, err := executer.CreateExecuter(r.ctx, param)
	if err != nil {
		r.Error().Msgf("param.TaskName(%s) create fail:%s\n", param.Executer.TaskName, err)
		return err
	}

	if err := e.Run(); err != nil {
		return err
	}
	r.exec.Store(param.Executer.TaskName, e)
	return nil
}

func (r *Runtime) runCrudCmd(conn *websocket.Conn, param *model.Param) (err error) {

	var tm cronex.TimerNoder
	switch {
	case param.IsCreate():
		tm, err = r.cron.AddFunc(param.Trigger.Cron, func() {
			// 创建执行器
			err = r.createToExec(param)
			// TODO 错误要上报到gate模块
		})

	case param.IsRemove(), param.IsStop():
		// 删除和stop对于runtime是一样，停止当前运行的，然后从sync.Map删除
		err = r.removeFromExec(param)
	case param.IsUpdate():
		// 先删除
		if err = r.removeFromExec(param); err != nil {
			return err
		}
		err = r.createToExec(param)
	}
	return err
}

// 接受来自gate服务的命令, 执行并返回结果
func (r *Runtime) readLoop(conn *websocket.Conn) error {

	var param model.Param

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
		err := conn.ReadJSON(&param) //这里不加超时时间, 一直监听gate推过来的信息
		if err != nil {
			return err
		}

		go func() {
			if err := r.runCrudCmd(conn, &param); err != nil {
				r.Error().Msgf("runtime.runCrud:%s\n", err)
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

		addrs := r.keyAddr()
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
