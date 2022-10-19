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
	ltaskPath := model.RuntimeNodeToLocalTaskPath(runtimeNode, taskName)
	_, err = defaultKVC.Put(m.ctx, ltaskPath, model.CanRun)
	return err
}

// 广播任务
func (m *Mjobs) broadcast(taskName string, param *model.Param) (err error) {
	m.runtimeNode.Range(func(key, val any) bool {
		ltaskPath := model.RuntimeNodeToLocalTaskPath(key.(string), taskName)
		// 向本地队列写入任务
		_, err = defaultKVC.Put(m.ctx, ltaskPath, model.CanRun)

		return err == nil
	})

	return err
}

// watch runtime node的变化
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
				//m.runtimeNode.Store(string(ev.Kv.Key), string(ev.Kv.Value))
				m.Debug().Msgf("Is this key() modified??? Not expected\n", string(ev.Kv.Key))
			case ev.Type == clientv3.EventTypeDelete:
				// 被删除
				// TODO 重新分配队列
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

	// 创建分布式锁
	l := concurrency.NewMutex(s, model.AssignTaskMutex)
	all := make([]string, 0, len(oneTask))

	l.Lock(ctx)
	defer l.Unlock(ctx)

	var runtimeNodes []string
	m.runtimeNode.Range(func(key, value any) bool {
		runtimeNodes = append(runtimeNodes, key.(string))
		return true
	})

	for kv := range oneTask {
		// 从状态信息里面获取tastName
		taskName := model.TaskNameFromStatePath(kv.key)
		if taskName == "" {
			m.Debug().Msgf("taskName is empty, %s\n", kv.key)
			continue
		}
		// 查看这个task的状态
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
		// 可以直接执行的taskName
		all = append(all, taskName)

		for _, taskName := range all {

			rsp, err := defaultKVC.Get(m.ctx, model.FullGlobalTaskPath(taskName))
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
	m.watchGlobalTaskState()
}
