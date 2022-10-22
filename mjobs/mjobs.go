package mjobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
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
	EtcdAddr  []string      `clop:"short;long;greedy" usage:"etcd address" valid:"required"`
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

func newKv(key string, value string) kv {
	return kv{key: key, val: value}
}

func (m *Mjobs) watchGlobalTaskState() {
	// 先获取节点状态
	rsp, err := defaultKVC.Get(m.ctx, model.GlobalTaskPrefixState, clientv3.WithPrefix(), clientv3.WithFilterDelete())
	if err == nil {
		for _, e := range rsp.Kvs {
			key := string(e.Key)
			value := string(e.Value)

			state, err := model.ValueToState(e.Value)
			if err != nil {
				m.Error().Msgf("watchGlobalTaskState, value to state %s\n", err)
				continue
			}

			if state.IsCanRun() {
				m.taskChan <- newKv(key, value)
			}
		}
	}

	rev := rsp.Header.Revision + 1
	readGlobal := defautlClient.Watch(m.ctx, model.GlobalTaskPrefixState, clientv3.WithPrefix(), clientv3.WithRev(rev))
	for ersp := range readGlobal {
		for _, ev := range ersp.Events {
			key := string(ev.Kv.Key)
			value := string(ev.Kv.Value)
			switch {
			case ev.IsCreate():
				m.Debug().Msgf("create global task:%s, state:%s\n", ev.Kv.Key, ev.Kv.Value)
				m.taskChan <- newKv(key, value)

			case ev.IsModify():
				m.Debug().Msgf("update global task:%s, state:%s\n", ev.Kv.Key, ev.Kv.Value)
				m.taskChan <- newKv(key, value)

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
				go func(oneTask []kv) {

					defer func() {
						if err := recover(); err != nil {
							m.Error().Msgf("fail, %s\n", err)
							fmt.Printf("%s\n", debug.Stack())
						}
					}()
					m.assign(oneTask, model.AssignTaskMutex, false)

				}(oneTask)

				oneTask = createOneTask()
			}
			run = false
		}
	}
}

func (m *Mjobs) setTaskToLocalrunq(taskName string, param *model.Param, runtimeNodes []string, failover bool) (err error) {
	if param.IsOneRuntime() {
		err = m.oneRuntime(taskName, param, runtimeNodes, failover)
	} else if param.IsBroadcast() {
		err = m.broadcast(taskName, param)
	} else {
		m.Warn().Msgf("Unknown kind:%s\n", param.Kind)
	}
	return err
}

// 执行单机任务
// 如果是create的任务，或者failover故障转移的任务，任选一个runtime执行
// 如果是Stop, Delete, update 还是由原先的runtime执行
func (m *Mjobs) oneRuntime(taskName string, param *model.Param, runtimeNodes []string, failover bool) (err error) {
	runtimeNode := utils.SliceRandOne(runtimeNodes)
	rsp, err := defaultKVC.Get(m.ctx, model.FullGlobalTaskState(taskName))
	if err != nil {
		m.Error().Msgf("oneRuntime %s\n", err)
		return err
	}
	if !failover && !param.IsCreate() {
		state, err := model.ValueToState(rsp.Kvs[0].Value)
		if err != nil {
			m.Error().Msgf("oneRuntime ValueToState %s\n", err)
			return err
		}

		m.Debug().Msgf("state:%v\n", state)
		runtimeNode = state.RuntimeNode
	}

	if len(runtimeNode) == 0 {
		m.Warn().Msgf("The runtimeNode is expected to be valuable\n")
		return
	}

	// 生成本地队列的名字
	ltaskPath := model.RuntimeNodeToLocalTask(runtimeNode, taskName)
	// 向本地队列写入任务
	if _, err = defaultKVC.Put(m.ctx, ltaskPath, model.CanRun); err != nil {
		return err
	}

	fullTaskState := model.FullGlobalTaskState(taskName)
	// 获取全局队列中的状态
	rsp, err = defaultKVC.Get(m.ctx, fullTaskState)
	if err != nil {
		return err
	}

	// 更新状态中的runtimeNode
	newValue, err := model.MarshalToJson(runtimeNode, model.Running)
	if err != nil {
		return err
	}

	txn := defaultKVC.Txn(m.ctx)
	txnRsp, err := txn.If(
		clientv3.Compare(clientv3.ModRevision(fullTaskState), "=", rsp.Kvs[0].ModRevision),
	).Then(
		clientv3.OpPut(fullTaskState, string(newValue)),
	).Commit()

	if err != nil {
		return err
	}

	if !txnRsp.Succeeded {
		err = errors.New("Transaction execution failed")
	}

	// 更新状态中的值
	m.Debug().Msgf("oneRuntime:key(%s):value(%s)\n", model.FullGlobalTaskState(taskName), runtimeNode)
	return err
}

// 广播任务
func (m *Mjobs) broadcast(taskName string, param *model.Param) (err error) {
	m.runtimeNode.Range(func(key, val any) bool {
		ltaskPath := model.RuntimeNodeToLocalTask(key.(string), taskName)
		// 向本地队列写入任务
		// TODO 事务
		_, err = defaultKVC.Put(m.ctx, ltaskPath, model.CanRun)
		defaultKVC.Put(m.ctx, model.FullGlobalTaskState(taskName), model.RunningJSON)

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
			key := string(e.Key)
			val := string(e.Value)
			m.runtimeNode.Store(key, val)
		}
	}

	rev := rsp.Header.Revision + 1
	// watch节点后续变化
	runtimeNode := defautlClient.Watch(m.ctx, model.RuntimeNodePrefix, clientv3.WithPrefix(), clientv3.WithRev(rev))
	for ersp := range runtimeNode {
		for _, ev := range ersp.Events {
			m.Debug().Msgf("mjobs.runtimeNodes key(%s) value(%s) create(%t), update(%t), delete(%t)\n",
				string(ev.Kv.Key), string(ev.Kv.Value), ev.IsCreate(), ev.IsModify(), ev.Type == clientv3.EventTypeDelete)
			switch {
			case ev.IsCreate():
				m.runtimeNode.Store(string(ev.Kv.Key), string(ev.Kv.Value))
				// 在当前内存中
			case ev.IsModify():
				// 这里不应该发生
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
		m.assign(oneTask, model.AssignTaskMutex, true)

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
		m.assign(oneTask, model.AssignTaskMutex, true)
	}
	return nil
}

// 分配任务的逻辑，使用分布式锁
func (m *Mjobs) assign(oneTask []kv, mutexName string, failover bool) {
	m.Debug().Msgf("call assign, oneTask.size:%d\n", len(oneTask))
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

	if len(runtimeNodes) == 0 {
		m.Warn().Msgf("assign.runtimeNodes.size is 0\n")
		return
	}

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

		if len(rsp.Kvs) > 0 {
			// 过滤正在运行中的任务
			val := rsp.Kvs[0].Value
			state, err := model.ValueToState(val)
			if err != nil {
				m.Error().Msgf("value to state%s\n", err)
				continue
			}

			if !state.IsCanRun() {
				continue
			}
		}
		// 可以直接执行的taskName
		all = append(all, taskName)

		for _, taskName := range all {

			m.Debug().Msgf("assign, taskName %s\n", taskName)
			rsp, err := defaultKVC.Get(m.ctx, model.FullGlobalTask(taskName))
			if err != nil {
				m.Error().Msgf("get global task path fail:%s\n", err)
				continue
			}

			if len(rsp.Kvs) == 0 {
				m.Warn().Msgf("get %s value is nil\n", model.FullGlobalTask(taskName))
				continue
			}
			param := model.Param{}
			err = json.Unmarshal(rsp.Kvs[0].Value, &param)
			if err != nil {
				m.Error().Msgf("Unmarshal global task fail:%s\n", err)
				continue
			}

			if err = m.setTaskToLocalrunq(taskName, &param, runtimeNodes, failover); err != nil {
				m.Error().Msgf("set task to local runq fail:%s\n", err)
				continue
			}
		}
	}
}

// mjobs子命令的的入口函数
func (m *Mjobs) SubMain() {
	if err := m.init(); err != nil {
		m.Error().Msgf("init:%s\n", err)
		return
	}

	go m.watchRuntimeNode()
	go m.taskLoop()
	m.watchGlobalTaskState()
}
