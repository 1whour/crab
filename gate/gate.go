package gate

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/slog"
	"github.com/gnh123/scheduler/utils"
	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"
)

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

	log *slog.Slog
	ctx context.Context
}

// 如果当前
var (
	conns         sync.Map
	defautlClient *clientv3.Client
)

func (r *Gate) init() error {

	r.ctx = context.TODO()
	r.log = slog.New(os.Stdout).SetLevel(r.Level)
	if r.Name == "" {
		r.Name = uuid.New().String()
	}

	r.newEtcdClient() //初始etcd客户端
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

	leaseID, err := utils.NewLeaseWithKeepalive(r.ctx, r.log, defautlClient, r.LeaseTime)
	if err != nil {
		return err
	}

	_, err = defautlClient.Put(r.ctx, r.genEtcdPath(), clientv3.WithLease(leaseID))
	return err
}

// 创建etcd的连接池
func (r *Gate) newEtcdClient() error {

	var err error
	defautlClient, err = clientv3.New(clientv3.Config{
		//Endpoints:   []string{"localhost:2379", "localhost:22379", "localhost:32379"},
		Endpoints:   r.EtcdAddr,
		DialTimeout: 5 * time.Second,
	})

	return err
}

// 该模块入口函数
func (r *Gate) SubMain() {
	if err := r.init(); err != nil {
		panic(err.Error())
	}
}
