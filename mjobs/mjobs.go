package mjobs

import (
	"context"
	"os"
	"time"

	"github.com/1whour/ktuo/model"
	"github.com/1whour/ktuo/slog"
	"github.com/1whour/ktuo/store/etcd"
	"github.com/1whour/ktuo/utils"
	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Mjobs模块定位是管理jobs
// 1.从全局队列里面分配任务到本地队列
// 2.监听runtime节点变化
// 3.如果runtime挂掉，把任务重新打包再分发，故障转移
// 4.进程重启时，加载任务到本地队列

// mjobs管理task
type Mjobs struct {
	EtcdAddr  []string      `clop:"short;long;greedy" usage:"etcd address" valid:"required"`
	NodeName  string        `clop:"short;long" usage:"node name"`
	Level     string        `clop:"short;long" usage:"log level"`
	LeaseTime time.Duration `clop:"long" usage:"lease time" default:"10s"`

	*slog.Slog
	ctx context.Context

	runtimeNode model.RuntimeNode
}

var (
	defautlClient *clientv3.Client
	defaultKVC    clientv3.KV
	defaultStore  *etcd.EtcdStore
)

// 初始化
func (m *Mjobs) init() (err error) {

	m.ctx = context.TODO()
	if m.NodeName == "" {
		m.NodeName = uuid.New().String()
	}
	m.Slog = slog.New(os.Stdout).SetLevel(m.Level).Str("mjobs", m.NodeName)

	if defautlClient, err = utils.NewEtcdClient(m.EtcdAddr); err != nil { //初始etcd客户端
		return err
	}

	defaultKVC = clientv3.NewKV(defautlClient) // 内置自动重试的逻辑
	defaultStore, err = etcd.NewStore(m.EtcdAddr, m.Slog, &m.runtimeNode)
	return err
}

func (m *Mjobs) SubMain() {
	if err := m.init(); err != nil {
		m.Error().Msgf("init:%s\n", err)
		return
	}

	// 异常恢复逻辑
	go m.restartRunning()
	// 监控runtime节点消失的
	go m.watchRuntimeNode()
	m.watchGlobalTaskState()
}
