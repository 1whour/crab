package runtime

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gnh123/scheduler/executer"
	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/slog"
	"github.com/gnh123/scheduler/utils"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/guonaihong/gstl/cmp"
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
type Runtime struct {
	EtcdAddr     []string      `clop:"short;long" usage:"etcd address"`
	GateAddr     []string      `clop:"long" usage:"endpoint address"`
	Level        string        `clop:"long" usage:"log level" default:"error"`
	WriteTimeout time.Duration `clop:"short;long" usage:"Timeout when writing messages"`
	Name         string        `clop:"short;long" usage:"node name"`
	ctx          context.Context
	*slog.Slog

	muWc sync.Mutex //保护多个go程写同一个conn

	sync.RWMutex
	exec  sync.Map
	addrs map[string]string
}

func (r *Runtime) init() (err error) {
	rand.Seed(time.Now().UnixNano())

	// 如果节点名为空，默认给个uuid
	if r.Name == "" {
		r.Name = uuid.New().String()
	}

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

	readGateNode := defautlClient.Watch(r.ctx, model.GateNodePrefix, clientv3.WithPrefix())
	for ersp := range readGateNode {
		for _, ev := range ersp.Events {
			switch {
			case ev.IsCreate():
				// 把新的gate地址加到当前addrs里面
				r.Lock()
				r.addrs[string(ev.Kv.Value)] = string(ev.Kv.Key)
				r.Unlock()
			case ev.IsModify():
				// 更新addrs里面的状态
				r.Lock()
				r.addrs[string(ev.Kv.Value)] = string(ev.Kv.Key)
				r.Unlock()
			case ev.Type == clientv3.EventTypeDelete:
				// 把被删除的gate从当前addrs里面移除
				r.Lock()
				delete(r.addrs, string(ev.Kv.Value))
				r.Unlock()
			}
		}
	}
}

// 写回错误的结果
func (r *Runtime) writeError(conn *websocket.Conn, to time.Duration, code int, msg string) (err error) {
	r.muWc.Lock()
	err = utils.WriteJsonTimeout(conn, model.RuntimeResp{Code: code, Message: msg}, to)
	r.muWc.Lock()
	return err
}

// 写回正确的结果
func (r *Runtime) writeResult(conn *websocket.Conn, to time.Duration, result string) (err error) {
	// 可能有多个go程写conn，加锁保护下
	r.Lock()
	err = utils.WriteJsonTimeout(conn, model.RuntimeResp{Result: result}, to)
	r.Unlock()
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

func (r *Runtime) runCrud(conn *websocket.Conn, param *model.Param) (err error) {

	switch {
	case param.IsCreate():
		// 创建执行器
		err = r.createToExec(param)
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

	for {
		err := conn.ReadJSON(&param) //这里不加超时时间, 一直监听gate推过来的信息
		if err != nil {
			return err
		}

		go func() {
			if err := r.runCrud(conn, &param); err != nil {
				r.writeError(conn, r.WriteTimeout, 1, err.Error())
				return
			}
		}()
	}

}

// 创建一个长连接
func (r *Runtime) createConntion(gateAddr string) error {

	c, _, err := websocket.DefaultDialer.Dial(gateAddr+"/"+model.TASK_STREAM_URL, nil)
	if err != nil {
		r.Error().Msgf("runtime:dial:%s\n", err)
		return err
	}

	defer c.Close()
	err = utils.WriteJsonTimeout(c, model.RuntimeWhoami{Name: r.Name}, r.WriteTimeout)
	if err != nil {
		return err
	}

	go func() {}()
	return r.readLoop(c)
}

// 初始化时创建 只创建一个长连接
// 故意这么设计
// 为了简化gate广播发送的逻辑, 一个runtime只会连一个gate，并且只有一个长连接，
// 这样不需要考虑去重，引入额外中间件，简化设计
func (r *Runtime) createConnRand() {

	interval := intervalTime

	for {

		r.Lock()
		addrs := make([]string, 0, len(r.addrs))
		for _, ip := range addrs {
			addrs = append(addrs, ip)
		}
		r.Unlock()

		sort.Strings(addrs)
		index := rand.Int31n(int32(len(addrs)))

		for i := 0; i < 2; i++ {

			if err := r.createConntion(addrs[index]); err != nil {
				// 如果握手或者上传第一个包失败，sleep 下，再重连一次
				time.Sleep(interval)
				interval *= 2
				interval = cmp.Min(interval, maxIntervalTime)
				r.Error().Msgf("createConnection fail:%v\n", err)
			} else {
				interval = intervalTime
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
