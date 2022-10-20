package mjobs

import (
	"context"
	"encoding/json"
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
	EtcdAddr  []string      `clop:"short;long;greedy" usage:"etcd address"`
	NodeName  string        `clop:"short;long" usage:"node name"`
	Level     string        `clop:"short;long" usage:"log level"`
	LeaseTime time.Duration `clop:"long" usage:"lease time" default:"10s"`

	*slog.Slog
	ctx      context.Context
	taskChan chan kv

	runtimeNode sync.Map
}

var (
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
				m.Debug().Msgf("create global task:%s, state:%s\n", ev.Kv.Key, ev.Kv.Value)
				m.taskChan <- kv{key: string(ev.Kv.Key), val: string(ev.Kv.Value)}

			case ev.IsModify():
				m.Debug().Msgf("update global task:%s, state:%s\n", ev.Kv.Key, ev.Kv.Value)
				m.taskChan <- kv{key: string(ev.Kv.Key), val: string(ev.Kv.Value)}

			case ev.Type == clientv3.EventTypeDelete:
				m.Debug().Msgf("delete global task:%s, state:%s\n", ev.Kv.Key, ev.Kv.Value)
			}
		}
	}
}

var createOneTask = func() []kv {
	return make([]kv, 0, 100)
}

// 从watch里面读取创建任务，当任务满足一定条件，比如满足一定条数，满足一定时间
// 先获取分布式锁，然后把任务打散到对应的gate节点，该节点负责推送到runtime节点
func (m *Mjobs) taskLoop() {

	tk := time.NewTicker(time.Second)

	oneTask := createOneTask()
	run := false
	for {

		select {
		case t := <-m.taskChan:
			oneTask = append(oneTask, t)
			if len(oneTask) == cap(oneTask) {
				run = true
				goto proc
			}
		case <-tk.C:
			run = true
			//或者超时
			goto proc
		}

	proc:
		if run {
			if len(oneTask) > 0 {
				go m.assign(oneTask, model.AssignTaskMutex)
				oneTask = createOneTask()
			}
			run = false
		}
	}
}

func (m *Mjobs) setTaskToLocalrunq(taskName string, param *model.Param, runtimeNodes []string) (err error) {
	if param.IsOneRuntime() {
		err = m.oneRuntime(taskName, param, runtimeNodes)
	} else if param.IsBroadcast() {
		err = m.broadcast(taskName, param)
	} else {
		m.Warn().Msgf("Unknown kind:%s\n", param.Kind)
	}
	return err
}

// 单机任务
func (m *Mjobs) oneRuntime(taskName string, param *model.Param, runtimeNodes []string) (err error) {
	runtimeNode := utils.SliceRandOne(runtimeNodes)
	// 向本地队列写入任务
	ltaskPath := model.RuntimeNodeToLocalTask(runtimeNode, taskName)
	_, err = defaultKVC.Put(m.ctx, ltaskPath, model.CanRun)
	return err
}

// 广播任务
func (m *Mjobs) broadcast(taskName string, param *model.Param) (err error) {
	m.runtimeNode.Range(func(key, val any) bool {
		ltaskPath := model.RuntimeNodeToLocalTask(key.(string), taskName)
		// 向本地队列写入任务
		_, err = defaultKVC.Put(m.ctx, ltaskPath, model.CanRun)

		return err == nil
	})

	return err
}

// watch runtime node的变化, 把node信息同步到内存里面
func (m *Mjobs) watchRuntimeNode() {
	// 先一次性获取当前节点
	rsp, err := defaultKVC.Get(m.ctx, model.RuntimeNodePrefix, clientv3.WithPrefix())
	if err == nil {
		for _, e := range rsp.Kvs {
			m.runtimeNode.Store(string(e.Key), string(e.Value))
		}
	}

	// watch节点后续变化
	// TODO，过滤版本
	runtimeNode := defautlClient.Watch(m.ctx, model.RuntimeNodePrefix, clientv3.WithPrefix())
	for ersp := range runtimeNode {
		for _, ev := range ersp.Events {
			switch {
			case ev.IsCreate():
				m.runtimeNode.Store(string(ev.Kv.Key), string(ev.Kv.Value))
				// 在当前内存中
			case ev.IsModify():
				// 这里不应该发生
				m.Warn().Msgf("Is this key() modified??? Not expected\n", string(ev.Kv.Key))
			case ev.Type == clientv3.EventTypeDelete:
				// 被删除
				go func() {
					if err := m.failover(string(ev.Kv.Key)); err != nil {
						m.Warn().Msgf("Is this key() modified??? Not expected\n", string(ev.Kv.Key))
					}
				}()
				m.runtimeNode.Delete(string(ev.Kv.Key))
			}
		}
	}
}

// 故障转移
// 监听runtime node的存活，如果死掉，把任务重新分发下
// 1.如果是故障转移的广播任务, 按道理，只应该在没有的机器上创建这个任务, TODO优化
// 2.如果是单runtime任务，任选一个runtime执行

func (m *Mjobs) failover(fullRuntime string) error {
	localPrefix := model.RuntimeNodeToLocalTaskPrefix(fullRuntime)
	rsp, err := defaultKVC.Get(m.ctx, localPrefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	oneTask := createOneTask()
	for _, keyval := range rsp.Kvs {
		oneTask = append(oneTask, kv{key: string(keyval.Key), val: string(keyval.Value)})
		m.assign(oneTask, model.AssignTaskMutex)

		for _, v := range oneTask {
			_, err := defaultKVC.Delete(m.ctx, v.key)
			if err != nil {
				m.Warn().Msgf("failover.delete fail:%s\n", err)
				continue
			}

		}
		oneTask = createOneTask()
	}

	if len(oneTask) > 0 {
		m.assign(oneTask, model.AssignTaskMutex)
	}
	return nil
}

// 分配任务的逻辑，使用分布式锁
func (m *Mjobs) assign(oneTask []kv, mutexName string) {

	s, _ := concurrency.NewSession(defautlClient)
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()

	// 创建分布式锁
	l := concurrency.NewMutex(s, mutexName)
	all := make([]string, 0, len(oneTask))

	l.Lock(ctx)
	defer l.Unlock(ctx)

	var runtimeNodes []string
	m.runtimeNode.Range(func(key, value any) bool {
		runtimeNodes = append(runtimeNodes, key.(string))
		return true
	})

	for _, kv := range oneTask {
		// 从状态信息里面获取tastName
		taskName := model.TaskNameFromState(kv.key)
		if taskName == "" {
			m.Debug().Msgf("taskName is empty, %s\n", kv.key)
			continue
		}
		// 查看这个task的状态
		statePath := model.FullGlobalTaskState(taskName)
		rsp, err := defaultKVC.Get(m.ctx, statePath)
		if err != nil {
			m.Error().Msgf("get global task state %s, path %s\n", err, statePath)
			continue
		}

		// 过滤正在运行中的任务
		val := string(rsp.Kvs[0].Value)
		if val == model.Running || val == model.Stop {
			continue
		}
		// 可以直接执行的taskName
		all = append(all, taskName)

		for _, taskName := range all {

			rsp, err := defaultKVC.Get(m.ctx, model.FullGlobalTask(taskName))
			if err != nil {
				m.Error().Msgf("get global task path fail:%s\n", err)
				continue
			}

			param := model.Param{}
			err = json.Unmarshal(rsp.Kvs[0].Value, &param)
			if err != nil {
				m.Error().Msgf("Unmarshal global task fail:%s\n", err)
				continue
			}

			if err = m.setTaskToLocalrunq(taskName, &param, runtimeNodes); err != nil {
				m.Error().Msgf("set task to local runq fail:%s\n", err)
				continue
			}
		}
	}
}

// mjobs子命令的的入口函数
func (m *Mjobs) SubMain() {
	m.init()
	go m.watchRuntimeNode()
	go m.taskLoop()
	m.watchGlobalTaskState()
}
