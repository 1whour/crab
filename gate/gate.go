package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/slog"
	"github.com/gnh123/scheduler/utils"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var upgrader = websocket.Upgrader{}

// TODO, 规范下错误码

// Gate模块定位是网关
// 1.注册自己的信息至etcd中
// 2.维护lambda过来的长连接
// 3.维护管理接口，保存到数据库中
type Gate struct {
	ServerAddr   string        `clop:"short;long" usage:"server address"`
	AutoFindAddr bool          `clop:"short;long" usage:"Automatically find unused ip:port, Only takes effect when ServerAddr is empty"`
	EtcdAddr     []string      `clop:"short;long;greedy" usage:"etcd address" valid:"required"`
	NamePrefix   string        `clop:"long" usage:"name prfix"`
	Name         string        `clop:"short;long" usage:"The name of the gate. If it is not filled, the default is uuid"`
	Level        string        `clop:"short;long" usage:"log level" default:"error"`
	LeaseTime    time.Duration `clop:"long" usage:"lease time" default:"4s"`
	WriteTime    time.Duration `clop:"long" usage:"write timeout" default"4s"`

	*slog.Slog
	ctx context.Context
}

func (g *Gate) NodeName() string {
	return fmt.Sprintf("%s-%s", g.NamePrefix, g.Name)
}

var (
	conns         sync.Map
	defautlClient *clientv3.Client
	defaultKVC    clientv3.KV
)

func (r *Gate) init() (err error) {

	r.getAddress()
	r.ctx = context.TODO()
	r.Slog = slog.New(os.Stdout).SetLevel(r.Level)
	if r.Name == "" {
		r.Name = uuid.New().String()
	}

	if r.LeaseTime < model.RuntimeKeepalive {
		r.LeaseTime = model.RuntimeKeepalive + time.Second
	}

	if defautlClient, err = utils.NewEtcdClient(r.EtcdAddr); err != nil { //初始etcd客户端
		return err
	}

	defaultKVC = clientv3.NewKV(defautlClient) // 内置自动重试的逻辑
	return nil
}

func (r *Gate) autoNewAddr() (addr string) {

	if r.AutoFindAddr {
		r.ServerAddr = utils.GetUnusedAddr()
		return r.ServerAddr
	}
	return
}

// 从ServerAddr获取，或者自动生成一个port
func (r *Gate) getAddress() string {
	if r.ServerAddr != "" {
		return r.ServerAddr
	}

	return r.autoNewAddr()
}

// gate的地址
// model.GateNodePrefix 注册到/scheduler/gate/node/gate_name
func (r *Gate) registerGateNode() error {
	addr := r.ServerAddr
	if addr == "" {
		r.Error().Msgf("The service startup address is empty, please set -s ip:port")
		os.Exit(1)
	}

	leaseID, err := utils.NewLeaseWithKeepalive(r.ctx, r.Slog, defautlClient, r.LeaseTime)
	if err != nil {
		return err
	}

	// 注册自己的节点信息
	_, err = defautlClient.Put(r.ctx, model.FullGateNode(r.NodeName()), addr, clientv3.WithLease(leaseID))
	return err
}

// 注册runtime节点，并负责节点lease的续期
func (r *Gate) registerRuntimeWithKeepalive(runtimeName string, keepalive chan bool) error {
	lease, leaseID, err := utils.NewLease(r.ctx, r.Slog, defautlClient, r.LeaseTime)
	if err != nil {
		r.Error().Msgf("registerRuntimeWithKeepalive.NewLease fail:%s\n", err)
		return err
	}
	// 注册runtime绑定的gate

	// 注册自己的节点信息
	_, err = defautlClient.Put(r.ctx, model.FullRuntimeNode(r.NodeName()), r.ServerAddr, clientv3.WithLease(leaseID))
	for range keepalive {
		lease.KeepAliveOnce(r.ctx, leaseID)
	}

	return nil
}

func (r *Gate) watchLocalRunq(runtimeName string, conn *websocket.Conn) {

	localTask := defautlClient.Watch(r.ctx, model.WatchLocalRuntimePrefix(runtimeName), clientv3.WithPrefix())

	for ersp := range localTask {
		for _, ev := range ersp.Events {

			localKey := string(ev.Kv.Key)
			taskName := model.TaskNameFromState(localKey)
			globalKey := model.FullLocalToGlobalTask(localKey)
			rsp, err := defaultKVC.Get(r.ctx, globalKey)
			if err != nil {
				r.Warn().Msgf("gate.watchLocalRunq: get param %s\n", err)
				continue
			}

			value := rsp.Kvs[0].Value

			var param model.Param
			err = json.Unmarshal(value, &param)
			if err != nil {
				r.Warn().Msgf("gate.watchLocalRunq:%s\n", err)
				continue
			}

			switch {
			case ev.IsCreate(), ev.IsModify():
				if err := utils.WriteMessageTimeout(conn, value, r.WriteTime); err != nil {
					r.Warn().Msgf("gate.watchLocalRunq, WriteMessageTimeout :%s\n", err)
					continue
				}

				if param.IsRemove() {
					defaultKVC.Delete(r.ctx, globalKey)
					defaultKVC.Delete(r.ctx, localKey)
					defaultKVC.Delete(r.ctx, model.FullGlobalTaskState(taskName)) //删除本地队列
				}
			case ev.Type == clientv3.EventTypeDelete:
				r.Debug().Msgf("delete global task:%s, state:%s\n", ev.Kv.Key, ev.Kv.Value)
			}
		}
	}
}

func (r *Gate) stream(c *gin.Context) {

	w := c.Writer
	req := c.Request

	con, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		r.Error().Msgf("upgrade:", err)
		return
	}
	defer con.Close()

	first := true
	keepalive := make(chan bool)
	for {
		// 读取心跳
		req := model.Whoami{}
		err := con.ReadJSON(&req)
		if err != nil {
			r.Warn().Msgf("gate.stream.read:%s\n", err)
			break
		}

		if first {
			go r.registerRuntimeWithKeepalive(req.Name, keepalive)
			go r.watchLocalRunq(req.Name, con)
			first = false
		} else {
			keepalive <- true
		}

		// TODO
		/*
			err = con.WriteJSON(mt, map[])
			if err != nil {
				log.Println("write:", err)
				break
			}
		*/
	}
}

func (r *Gate) ok(c *gin.Context, msg string) {
	r.Debug().Caller(1).Msg(msg)
	c.JSON(200, gin.H{"code": 0, "message": ""})
}

// 简单的包装函数
func (r *Gate) error(c *gin.Context, code int, format string, a ...any) {

	msg := fmt.Sprintf(format, a...)
	r.Error().Caller(1).Msg(msg)
	c.JSON(500, gin.H{"code": code, "message": msg})
}

// 把task信息保存至etcd
func (r *Gate) createTask(c *gin.Context) {
	var req model.Param
	err := c.ShouldBind(&req)
	if err != nil {
		r.error(c, 500, "createTask:%v, type:%s", err, c.ContentType())
		return
	}

	// 创建数据队列
	globalTaskName := model.FullGlobalTask(req.Executer.TaskName)
	// 创建状态队列
	globalTaskStateName := model.FullGlobalTaskState(req.Executer.TaskName)

	// 先get，如果有值直接返回
	rsp, err := defaultKVC.Get(r.ctx, globalTaskName, clientv3.WithKeysOnly())
	if len(rsp.Kvs) > 0 {
		r.error(c, 500, "duplicate creation:%s", globalTaskName)
		return
	}

	req.SetCreate() //设置action
	all, err := json.Marshal(req)
	if err != nil {
		r.error(c, 500, "marshal req:%v", err)
		return
	}

	txn := defaultKVC.Txn(r.ctx)
	txn.If(clientv3.Compare(clientv3.CreateRevision(globalTaskName), "=", 0)).
		Then(
			clientv3.OpPut(globalTaskName, string(all)),
			clientv3.OpPut(globalTaskStateName, model.CanRun),
		).Else()

	txnRsp, err := txn.Commit()
	if err != nil {
		r.error(c, 500, "事务执行失败err :%v", err)
		return
	}

	if !txnRsp.Succeeded {
		r.error(c, 500, "事务失败")
		return
	}

	r.ok(c, "createTask 执行成功") //返回正确业务码
}

// 删除etcd里面task信息，也直接下发命令更新runtime里面信息
func (r *Gate) deleteTask(c *gin.Context) {
	var req model.Param
	err := c.ShouldBind(&req)
	if err != nil {
		r.error(c, 500, "deleteTask:%v", err)
		return
	}

	// 生成数据队列的名字
	globalTaskName := model.FullGlobalTask(req.Executer.TaskName)
	// 生成状态队列的名字
	globalTaskStateName := model.FullGlobalTaskState(req.Executer.TaskName)

	// 先get，如果无值直接返回
	rsp, err := defaultKVC.Get(r.ctx, globalTaskName, clientv3.WithKeysOnly())
	if len(rsp.Kvs) == 0 {
		r.error(c, 500, "Task is empty and cannot be remove:%s", globalTaskName)
		return
	}

	req.SetRemove() //设置action为remove
	all, err := json.Marshal(req)
	if err != nil {
		r.error(c, 500, "marshal req:%v", err)
		return
	}

	txn := defaultKVC.Txn(r.ctx)
	txn.If(clientv3.Compare(clientv3.CreateRevision(globalTaskName), "=", rsp.Kvs[0].Version)).
		Then(
			clientv3.OpPut(globalTaskName, string(all)),
			clientv3.OpPut(globalTaskStateName, model.CanRun),
		).Else()

	txnRsp, err := txn.Commit()
	if err != nil {
		r.error(c, 500, "事务执行失败err :%v", err)
		return
	}

	if !txnRsp.Succeeded {
		r.error(c, 500, "事务失败")
		return
	}

	r.ok(c, "removeTask 执行成功") //返回正确业务码
}

// 更新etcd里面的task信息，也下发命令更新runtime里面信息
func (r *Gate) updateTask(c *gin.Context) {

	var req model.Param
	err := c.ShouldBind(&req)
	if err != nil {
		r.error(c, 500, "updateTask:%v", err)
		return
	}

	// 创建数据队列
	globalTaskName := model.FullGlobalTask(req.Executer.TaskName)
	// 创建状态队列
	globalTaskStateName := model.FullGlobalTaskState(req.Executer.TaskName)

	// 先get，如果有值直接返回
	rsp, err := defaultKVC.Get(r.ctx, globalTaskName, clientv3.WithKeysOnly())
	if len(rsp.Kvs) == 0 {
		r.error(c, 500, "Task is empty and cannot be updated:%s", globalTaskName)
		return
	}

	req.SetUpdate()

	all, err := json.Marshal(req)
	if err != nil {
		r.error(c, 500, "marshal req:%v", err)
		return
	}

	r.Debug().Msgf("get version:%v, ModRevision:%v\n", rsp.Kvs[0].Version, rsp.Kvs[0].ModRevision)

	txn := defaultKVC.Txn(r.ctx)
	txn.If(clientv3.Compare(clientv3.ModRevision(globalTaskName), "=", rsp.Kvs[0].ModRevision)).
		Then(
			clientv3.OpPut(globalTaskName, string(all)),
			clientv3.OpPut(globalTaskStateName, model.CanRun),
		).Else()

	txnRsp, err := txn.Commit()
	if err != nil {
		r.error(c, 500, "事务执行失败err :%v", err)
		return
	}

	if !txnRsp.Succeeded {
		r.error(c, 500, "事务失败")
		return
	}

	r.ok(c, "updateTask 执行成功") //返回正确业务码
}

// 更新etcd里面的task信息，置为静止，下发命令取消正在执行中的task
func (r *Gate) stopTask(c *gin.Context) {

	var req model.Param
	err := c.ShouldBind(&req)
	if err != nil {
		r.error(c, 500, "stopTask:%v", err)
		return
	}

	// 创建数据队列
	globalTaskName := model.FullGlobalTask(req.Executer.TaskName)
	// 创建状态队列
	globalTaskStateName := model.FullGlobalTaskState(req.Executer.TaskName)

	// 先get，如果有值直接返回
	rsp, err := defaultKVC.Get(r.ctx, globalTaskName, clientv3.WithKeysOnly())
	if len(rsp.Kvs) == 0 {
		r.error(c, 500, "Task is empty and cannot be stop:%s", globalTaskName)
		return
	}

	req.SetStop() //设置action为stop
	all, err := json.Marshal(req)
	if err != nil {
		r.error(c, 500, "marshal req:%v", err)
		return
	}

	txn := defaultKVC.Txn(r.ctx)
	txn.If(clientv3.Compare(clientv3.CreateRevision(globalTaskName), "=", rsp.Kvs[0].Version)).
		Then(
			clientv3.OpPut(globalTaskName, string(all)),
			clientv3.OpPut(globalTaskStateName, model.CanRun),
		).Else()

	txnRsp, err := txn.Commit()
	if err != nil {
		r.error(c, 500, "事务执行失败err :%v", err)
		return
	}

	if !txnRsp.Succeeded {
		r.error(c, 500, "事务失败")
		return
	}

	r.ok(c, "stopTask 执行成功") //返回正确业务码
}

// 该模块入口函数
func (r *Gate) SubMain() {
	if err := r.init(); err != nil {
		return
	}

	go r.registerGateNode()

	g := gin.New()
	g.GET(model.TASK_STREAM_URL, r.stream) //流式接口，主动推送任务至runtime
	g.POST(model.TASK_CREATE_URL, r.createTask)
	g.DELETE(model.TASK_DELETE_URL, r.deleteTask)
	g.PUT(model.TASK_UPDATE_URL, r.updateTask)
	g.POST(model.TASK_STOP_URL, r.stopTask)

	r.Debug().Msgf("gate:serverAddr:%s\n", r.ServerAddr)
	for i := 0; i < 3; i++ {
		if err := g.Run(r.ServerAddr); err != nil {
			r.autoNewAddr()
			r.Debug().Msgf("gate:serverAddr:%s\n", r.ServerAddr)
			time.Sleep(time.Millisecond * 500)
		}
	}
}
