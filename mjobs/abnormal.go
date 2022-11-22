package mjobs

import (
	"time"

	"github.com/gnh123/scheduler/model"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// 故障转移
// 监听runtime node的存活，如果死掉，把任务重新分发下
// 1.如果是故障转移的广播任务, 按道理，只应该在没有的机器上创建这个任务, 目前广播 TODO优化
// 2.如果是单runtime任务，任选一个runtime执行
func (m *Mjobs) failover(fullRuntime string) error {
	localPrefix := model.ToLocalTaskPrefix(fullRuntime)
	rsp, err := defaultKVC.Get(m.ctx, localPrefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, keyval := range rsp.Kvs {
		key := model.ToGlobalTaskState(string(keyval.Key))
		rsp, err := defaultKVC.Get(m.ctx, key)
		if len(rsp.Kvs) == 0 {
			continue
		}

		defaultStore.AssignMutex(m.ctx, model.KeyVal{Key: string(rsp.Kvs[0].Key), Val: string(rsp.Kvs[0].Value)}, true)

		_, err = defaultKVC.Delete(m.ctx, string(keyval.Key))
		if err != nil {
			m.Warn().Msgf("failover.delete fail:%s\n", err)
			continue
		}

	}

	return nil
}

// 异常条件:
// 1.mjobs模板把任务分配到本地队列状态是running, 如果这时候runtime挂了, runtimeNode会被从etcd自动删除
//  判断条件就是state.IsRunning() && len(state.restartRunning) > 0 (running和当前任务的runtimeNode不存在)

// 2.如果把任务写回至runtime websocket连接挂了，runtime进程也死了，可以走进第一个逻辑恢复, 1,2 算一种异常

// 3.如果gate把任务分配至runtime这时候连接挂了，runtime进程还在，需要把任务的状态修改(gate)为failed, 下一次重新分配

// 4.如果任务是Stop, Rm的任务仅仅保证执行一次

// 做法:
// 集群稳定的前提下(当runtime的个数>=1 gate的个数>=1)，什么样的任务可以被恢复?

// 1.如果是Create和Update的任务，任务绑定的runtime是空, State是任何状态，都需要被恢复, 这是一个还需要被运行的状态

// 2.如果是Stop和Rm的任务, 如果runtimeNode不为空。Number == 0时会尝试一次
func (m *Mjobs) needRestart(state model.State) bool {
	if state.IsFailed() {
		return true
	}

	var ip *clientv3.GetResponse
	var err error
	// runtimeNode为空
	if len(state.RuntimeNode) == 0 {
		goto next
	}

	ip, err = defaultKVC.Get(m.ctx, state.RuntimeNode)
	if err != nil {
		//m.Error().Msgf("restartRunning: get ip %v, key(%s)\n", err, state.RuntimeNode)
		goto next
	}

	if (state.IsStop() || state.IsRemove()) && state.InRuntime && len(ip.Kvs) > 0 {
		return true
	}

next:
	return (state.IsCreate() || state.IsUpdate()) && (ip == nil || len(ip.Kvs) == 0)
}

// 检查全局队列中的死任务Running状态的，重新加载到runtime里面
// 单机任务随机找个节点执行
// 广播任务, 只广播到当前的所有的节点
func (m *Mjobs) restartRunning() {

	for {
		// 先获取任务的前缀
		rsp, err := defaultKVC.Get(m.ctx, model.GlobalTaskPrefixState, clientv3.WithPrefix())
		if err != nil {
			m.Error().Msgf("restartRunning, get state prefix:%v\n", err)
			continue
		}

		// 遍历所有的全局任务
		for _, kv := range rsp.Kvs {
			state, err := model.ValueToState(kv.Value)
			if err != nil {
				m.Error().Msgf("restartRunning: value to state:%v\n", err)
				continue
			}

			if m.needRestart(state) {
				m.Debug().Msgf("restartRunning, need fix %s, state:%v\n", kv.Key, state)

				fullGlobalTask := string(kv.Key)
				taskName := model.TaskName(fullGlobalTask)
				ltaskPath := model.ToLocalTask(fullGlobalTask, taskName)

				var err error
				oneTask := model.KeyVal{Key: string(kv.Key),
					Val:     string(kv.Value),
					Version: int(kv.ModRevision),
					State:   state,
				}

				defaultStore.AssignMutexWithCb(m.ctx, oneTask, true, func() {
					_, err = defaultKVC.Delete(m.ctx, ltaskPath)
				})

				if err != nil {
					m.Error().Msgf("restartRunning, %v\n", err)
					continue
				}

			}
		}

		// 3s检查一次
		time.Sleep(time.Second * 5)
	}

}
