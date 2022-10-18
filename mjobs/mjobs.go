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
	EtcdAddr  []string      `clop:"short;long" usage:"etcd address"`
	NodeName  string        `clop:"short;long" usage:"node name"`
	Level     string        `clop:"short;long" usage:"log level"`
	LeaseTime time.Duration `clop:"long" usage:"lease time" default:"10s"`

	*slog.Slog
	ctx      context.Context
	taskChan chan kv

	runtimeNode sync.Map
}

var (
	conns         sync.Map
	defautlClient *clientv3.Client
	defaultKVC    clientv3.KV
)

func (m *Mjobs) init() (err error) {

	m.ctx = context.TODO()
	if m.NodeName == "" {
		m.NodeName = uuid.New().String()
	}
	m.Slog = slog.New(os.Stdout).SetLevel(m.Level).Str("mjobs", m.NodeName)
	m.taskChan = make(chan kv, 100)

	if defautlClient, err = utils.NewEtcdClient(m.EtcdAddr); err != nil { //初始etcd客户端
		return err
	}

	defaultKVC = clientv3.NewKV(defautlClient) // 内置自动重试的逻辑
	return nil
}

type kv struct {
	key string
	val string
}

func (m *Mjobs) watchGlobalTaskState() {

	readGateNode := defautlClient.Watch(m.ctx, model.GlobalTaskPrefixState, clientv3.WithPrefix())
	for ersp := range readGateNode {
		for _, ev := range ersp.Events {
			switch {
			case ev.IsCreate():
				m.taskChan <- kv{key: string(ev.Kv.Key), val: string(ev.Kv.Value)}
				// 创建新的read/write loop
			case ev.IsModify():
				m.Debug().Msgf("update global task:%s, state:%s\n", ev.Kv.Key, ev.Kv.Value)
			case ev.Type == clientv3.EventTypeDelete:
				m.Debug().Msgf("delete global task:%s, state:%s\n", ev.Kv.Key, ev.Kv.Value)
			}
		}
	}
}

// 从watch里面读取创建任务，当任务满足一定条件，比如满足一定条数，满足一定时间
// 先获取分布式锁，然后把任务打散到对应的gate节点，该节点负责推送到runtime节点
func (m *Mjobs) readLoop() {

	tk := time.NewTicker(time.Second)

	createOneTask := func() chan kv {
		return make(chan kv, 100)
	}

	oneTask := createOneTask()
	for {

		select {
		case t := <-m.taskChan:
			select {
			case oneTask <- t:
			default:
				goto proc
			}
		case <-tk.C:
			goto proc
		}

	proc:
		if false {
			if len(oneTask) > 0 {
				close(oneTask)
				go m.assign(oneTask)
				oneTask = createOneTask()
			}
		}
	}
}

// 单机任务
func (m *Mjobs) oneRuntime() {

}

// 广播任务
func (m *Mjobs) broadcast() {

}

func (m *Mjobs) watchRuntimeNode() {
	rsp, err := defaultKVC.Get(m.ctx, model.RuntineNodePrfix, clientv3.WithPrefix())
	if err == nil {
		for _, e := range rsp.Kvs {
			m.runtimeNode.Store(string(e.Key), string(e.Value))
		}
	}

	runtimeNode := defautlClient.Watch(m.ctx, model.RuntineNodePrfix, clientv3.WithPrefix())
	for ersp := range runtimeNode {
		for _, ev := range ersp.Events {
			switch {
			case ev.IsCreate():
				m.runtimeNode.Store(string(ev.Kv.Key), string(ev.Kv.Value))
				// 在当前内存中
			case ev.IsModify():
				//m.runtimeNode.Store(string(ev.Kv.Key), string(ev.Kv.Value))
				m.Debug().Msgf("Is this key() modified??? Not expected\n", string(ev.Kv.Key))
			case ev.Type == clientv3.EventTypeDelete:
				// 被删除
				m.runtimeNode.Delete(string(ev.Kv.Key))
			}
		}
	}
}

// 分配任务的逻辑，使用分布式锁
func (m *Mjobs) assign(oneTask chan kv) {

	s, _ := concurrency.NewSession(defautlClient)
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()
	l := concurrency.NewMutex(s, model.AssignTaskMutex)
	l.Lock(ctx)
	all := make([]kv, 0, len(oneTask))

	for kv := range oneTask {
		taskName := model.TaskNameFromStatePath(kv.key)
		if taskName == "" {
			m.Debug().Msgf("taskName is empty, %s\n", kv.key)
			continue
		}

		statePath := model.FullGlobalTaskStatePath(taskName)
		rsp, err := defaultKVC.Get(m.ctx, statePath)
		if err != nil {
			m.Error().Msgf("get global task state %s, path %s\n", err, statePath)
			continue
		}

		val := string(rsp.Kvs[0].Value)
		if val == model.Running || val == model.Stop {
			continue
		}
		all = append(all, taskName)
	}

	l.Unlock(ctx)
}

// mjobs子命令的的入口函数
func (m *Mjobs) SubMain() {
	m.init()
	go m.watchRuntimeNode()
	m.watchGlobalTaskState()
}
