package mjobs

import (
	"context"
	"encoding/json"
	"errors"
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

	runtimeNode sync.Map
}

type mParam struct {
	model.Param
	stateModRevision int
	dataVersion      int
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

	if defautlClient, err = utils.NewEtcdClient(m.EtcdAddr); err != nil { //初始etcd客户端
		return err
	}

	defaultKVC = clientv3.NewKV(defautlClient) // 内置自动重试的逻辑
	return nil
}

type KeyVal struct {
	key     string
	val     string
	state   model.State
	version int
}

var createOneTask = func() []KeyVal {
	return make([]KeyVal, 0, 100)
}

func (m *Mjobs) setTaskToLocalrunq(taskName string, param *mParam, runtimeNode string, failover bool) (err error) {
	if param.IsOneRuntime() {
		err = m.oneRuntime(taskName, param, runtimeNode, failover)
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
func (m *Mjobs) oneRuntime(taskName string, param *mParam, runtimeNode string, failover bool) (err error) {
	fullTaskState := model.FullGlobalTaskState(taskName)
	// 获取全局队列中的状态
	rsp, err := defaultKVC.Get(m.ctx, fullTaskState)
	if err != nil {
		m.Error().Msgf("oneRuntime %s\n", err)
		return err
	}

	// 如果是更新或者删除或者stop的任务, 找到绑定的runtimeNode
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

	// 生成本地队列的名字, 包含runtime和taskName
	ltaskPath := model.RuntimeNodeToLocalTask(runtimeNode, taskName)
	// 向本地队列写入任务
	if _, err = defaultKVC.Put(m.ctx, ltaskPath, model.CanRun); err != nil {
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
func (m *Mjobs) broadcast(taskName string, param *mParam) (err error) {
	m.runtimeNode.Range(func(key, val any) bool {
		err = m.oneRuntime(taskName, param, key.(string), false)
		return err == nil
	})

	return err
}

// 使用分布式锁
func (m *Mjobs) assignMutex(oneTask KeyVal, failover bool) {
	m.assignMutexWithCb(oneTask, failover, nil)
}

func (m *Mjobs) assignMutexWithCb(oneTask KeyVal, failover bool, cb func()) {
	mutexName := model.AssignTaskMutex(oneTask.key)

	s, _ := concurrency.NewSession(defautlClient)
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()

	// 创建分布式锁
	l := concurrency.NewMutex(s, mutexName)

	// 获取锁失败直接返回
	if err := l.TryLock(ctx); err != nil {
		m.Debug().Msgf("assign trylock:%s\n", err)
		return
	}
	defer l.Unlock(ctx)

	m.assign(oneTask, failover)
	if cb != nil {
		cb()
	}
}

// 随机选择一个runtimeNode
func (m *Mjobs) selectRuntimeNode() (string, error) {

	var runtimeNodes []string
	m.runtimeNode.Range(func(key, value any) bool {
		runtimeNodes = append(runtimeNodes, key.(string))
		return true
	})

	if len(runtimeNodes) == 0 {
		m.Warn().Msgf("assign.runtimeNodes.size is 0\n")
		return "", errors.New("assign.runtimeNodes.size is 0")
	}
	return utils.SliceRandOne(runtimeNodes), nil
}

// 分配任务的逻辑
func (m *Mjobs) assign(oneTask KeyVal, failover bool) {
	m.Debug().Msgf("call assign, key:%s, state:%s\n", oneTask.key, oneTask.state.State)

	kv := oneTask
	runtimeNode, err := m.selectRuntimeNode()
	if err != nil {
		return
	}
	// 从状态信息里面获取tastName
	taskName := model.TaskNameFromState(kv.key)
	if taskName == "" {
		m.Debug().Msgf("taskName is empty, %s\n", kv.key)
		return
	}

	m.Debug().Msgf("assign, taskName %s, action:%s\n", taskName)
	rsp, err := defaultKVC.Get(m.ctx, model.FullGlobalTask(taskName), clientv3.WithRev(int64(oneTask.version)))
	if err != nil {
		m.Error().Msgf("get global task path fail:%s\n", err)
		return
	}

	if len(rsp.Kvs) == 0 {
		m.Warn().Msgf("get %s value is nil\n", model.FullGlobalTask(taskName))
		return
	}

	param := mParam{}
	param.stateModRevision = kv.version
	param.dataVersion = int(rsp.Kvs[0].Version)
	err = json.Unmarshal(rsp.Kvs[0].Value, &param.Param)
	if err != nil {
		m.Error().Msgf("Unmarshal global task fail:%s\n", err)
		return
	}

	if err = m.setTaskToLocalrunq(taskName, &param, runtimeNode, failover); err != nil {
		m.Error().Msgf("set task to local runq fail:%s\n", err)
		return
	}
}

// mjobs子命令的的入口函数
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
