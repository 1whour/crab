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
	localPrefix := model.RuntimeNodeToLocalTaskPrefix(fullRuntime)
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

		m.assignMutex(KeyVal{key: string(rsp.Kvs[0].Key), val: string(rsp.Kvs[0].Value)}, true)

		_, err = defaultKVC.Delete(m.ctx, string(keyval.Key))
		if err != nil {
			m.Warn().Msgf("failover.delete fail:%s\n", err)
			continue
		}

	}

	return nil
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

			// 1.mjobs模板把任务分配到本地队列状态是running, 如果这时候runtime挂了
			//  判断条件就是state.IsRunning() && len(state.restartRunning) > 0

			// 2.如果把任务写回至runtime挂了，真挂了，可以走进第一个逻辑恢复, 这两种算一种异常

			// 3.如果gate把任务分配至runtime这时候连接挂了，runtime进程还在，需要把任务的状态修改(gate)为failed, 下一次重新分配

			// 4.如果任务是Stop, Rm的任务仅仅保证执行一次
			if state.IsRunning() || state.IsFailed() {
				ip, err := defaultKVC.Get(m.ctx, state.RuntimeNode)
				if err != nil {
					m.Error().Msgf("restartRunning: get ip %v\n", err)
					continue
				}

				if len(ip.Kvs) == 0 || state.IsFailed() {
					m.Debug().Msgf("restartRunning, get local.runq.task.runtimeNode.size:%d, need fix %s\n", len(ip.Kvs), kv.Key)

					fullGlobalTask := string(kv.Key)
					taskName := model.TaskNameFromGlobalTask(fullGlobalTask)
					ltaskPath := model.RuntimeNodeToLocalTask(fullGlobalTask, taskName)

					var err error
					oneTask := KeyVal{key: string(kv.Key),
						val:     string(kv.Value),
						version: int(kv.ModRevision),
						state:   state,
					}

					m.assignMutexWithCb(oneTask, true, func() {
						_, err = defaultKVC.Delete(m.ctx, ltaskPath)
					})

					if err != nil {
						m.Error().Msgf("restartRunning, %v\n", err)
						continue
					}

				}
			}
		}

		// 3s检查一次
		time.Sleep(time.Second * 5)
	}

}
