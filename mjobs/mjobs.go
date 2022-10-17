package mjobs

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/slog"
	"github.com/gnh123/scheduler/utils"
	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// mjobs管理task
type Mjobs struct {
	ServerAddr   string        `clop:"short;long" usage:"server address"`
	AutoFindAddr bool          `clop:"short;long" usage:"Automatically find unused ip:port, Only takes effect when ServerAddr is empty"`
	EtcdAddr     []string      `clop:"short;long" usage:"etcd address"`
	Name         string        `clop:"short;long" usage:"The name of the gate. If it is not filled, the default is uuid"`
	Level        string        `clop:"short;long" usage:"log level"`
	LeaseTime    time.Duration `clop:"long" usage:"lease time" default:"10s"`

	*slog.Slog
	ctx context.Context
}

var (
	conns         sync.Map
	defautlClient *clientv3.Client
	defaultKVC    clientv3.KV
)

func (m *Mjobs) init() (err error) {

	m.ctx = context.TODO()
	m.Slog = slog.New(os.Stdout).SetLevel(m.Level)
	if m.Name == "" {
		m.Name = uuid.New().String()
	}

	if defautlClient, err = utils.NewEtcdClient(m.EtcdAddr); err != nil { //初始etcd客户端
		return err
	}

	defaultKVC = clientv3.NewKV(defautlClient) // 内置自动重试的逻辑
	return nil
}

func (m *Mjobs) watchTask() {

	readGateNode := defautlClient.Watch(m.ctx, model.AllTaskPrefix, clientv3.WithPrefix())
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

// mjobs子命令的的入口函数
func (m *Mjobs) SubMain() {
	m.init()
	m.watchTask()
}
