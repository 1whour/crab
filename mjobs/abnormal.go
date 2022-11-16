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
		m.assignMutex(KeyVal{key: string(keyval.Key), val: string(keyval.Value)}, true)

		_, err := defaultKVC.Delete(m.ctx, string(keyval.Key))
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

			// TODO state.IsCanRun(), 这个状态是否能加入异常恢复
			if state.IsRunning() {
				ip, err := defaultKVC.Get(m.ctx, state.RuntimeNode)
				if err != nil {
					m.Error().Msgf("restartRunning: get ip %v\n", err)
					continue
				}

				if len(ip.Kvs) == 0 {
					m.Debug().Msgf("restartRunning, get runtimeNode.size:%d, need fix %s\n", len(ip.Kvs), kv.Key)

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
		time.Sleep(time.Second * 3)
	}

}
