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
	"go.etcd.io/etcd/client/v3/concurrency"
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
	ctx      context.Context
	taskChan chan string
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
	m.taskChan = make(chan string, 100)

	if defautlClient, err = utils.NewEtcdClient(m.EtcdAddr); err != nil { //初始etcd客户端
		return err
	}

	defaultKVC = clientv3.NewKV(defautlClient) // 内置自动重试的逻辑
	return nil
}

func (m *Mjobs) watchTask() {

	readGateNode := defautlClient.Watch(m.ctx, model.GlobalTaskPrefix, clientv3.WithPrefix())
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

// 从watch里面读取任务，当任务满足一定条件，比如满足一定条数，满足一定时间
// 先获取分布式锁，然后把任务打散到对应的gate节点，该节点负责推送到runtime节点
func (m *Mjobs) readLoop() {

	tk := time.NewTicker(time.Second)
	for {

		select {
		case <-m.taskChan:
		case <-tk.C:
		}
	}
}

// 分配任务的逻辑，使用分布式锁
func (m *Mjobs) assign() {
	s, _ := concurrency.NewSession(defautlClient)
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()
	l := concurrency.NewMutex(s, model.AssignTaskMutex)
	l.Lock(ctx)
	l.Unlock(ctx)
}

// mjobs子命令的的入口函数
func (m *Mjobs) SubMain() {
	m.init()
	m.watchTask()
}
