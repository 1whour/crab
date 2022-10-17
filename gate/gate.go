package gate

import (
	"context"
	"fmt"
	"log"
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

// Gate模块定位是网关
// 1.注册自己的信息至etcd中
// 2.维护lambda过来的长连接
// 3.维护管理接口，保存到数据库中
type Gate struct {
	ServerAddr   string        `clop:"short;long" usage:"server address"`
	AutoFindAddr bool          `clop:"short;long" usage:"Automatically find unused ip:port, Only takes effect when ServerAddr is empty"`
	EtcdAddr     []string      `clop:"short;long" usage:"etcd address"`
	Name         string        `clop:"short;long" usage:"The name of the gate. If it is not filled, the default is uuid"`
	Level        string        `clop:"short;long" usage:"log level"`
	LeaseTime    time.Duration `clop:"long" usage:"lease time" default:"10s"`

	*slog.Slog
	ctx context.Context
}

// 如果当前
var (
	conns         sync.Map
	defautlClient *clientv3.Client
)

func (r *Gate) init() (err error) {

	r.ctx = context.TODO()
	r.Slog = slog.New(os.Stdout).SetLevel(r.Level)
	if r.Name == "" {
		r.Name = uuid.New().String()
	}

	if defautlClient, err = utils.NewEtcdClient(r.EtcdAddr); err != nil { //初始etcd客户端
		return err
	}

	return nil
}

func (r *Gate) getAddress() string {
	if r.ServerAddr != "" {
		return r.ServerAddr
	}

	if r.AutoFindAddr {
		return utils.GetUnusedAddr()
	}
	return ""
}

func (r *Gate) genEtcdPath() string {
	return fmt.Sprintf("%s/%s", model.GateNodePrefix, r.Name)
}

// gate的地址
// 注册到/scheduler/gate/node/gate_name
func (r *Gate) register() error {
	addr := r.getAddress()
	if addr == "" {
		panic("The service startup address is empty, please set -s ip:port")
	}

	leaseID, err := utils.NewLeaseWithKeepalive(r.ctx, r.Slog, defautlClient, r.LeaseTime)
	if err != nil {
		return err
	}

	// 注册自己的节点信息
	_, err = defautlClient.Put(r.ctx, r.genEtcdPath(), r.ServerAddr, clientv3.WithLease(leaseID))
	return err
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

	for {
		// 读取执行结果，或者心跳
		mt, message, err := con.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}

		//log.Printf("recv: %s", message)
		err = con.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

// 把task信息保存至etcd
func (r *Gate) createTask(c *gin.Context) {

}

// 删除etcd里面task信息，也直接下发命令更新runtime里面信息
func (r *Gate) deleteTask(c *gin.Context) {

}

// 更新etcd里面的task信息，也下发命令更新runtime里面信息
func (r *Gate) updateTask(c *gin.Context) {

}

// 更新etcd里面的task信息，置为静止，下发命令取消正在执行中的task
func (r *Gate) cancelTask(c *gin.Context) {

}

// 该模块入口函数
func (r *Gate) SubMain() {
	if err := r.init(); err != nil {
		panic(err.Error())
	}

	g := gin.New()
	g.GET(model.TASK_STREAM_URL, r.stream) //流式接口，主动推送任务至runtime
	g.POST(model.TASK_CREATE_URL, r.createTask)
	g.DELETE(model.TASK_DELETE_URL, r.deleteTask)
	g.PUT(model.TASK_UPDATE_URL, r.updateTask)
	g.POST(model.TASK_CANCEL_URL, r.cancelTask)

	g.Run()
}
