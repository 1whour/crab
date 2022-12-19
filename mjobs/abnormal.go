package mjobs

import (
	"time"

	"github.com/1whour/crab/model"
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

		state, err := model.ValueToState(rsp.Kvs[0].Value)
		if err != nil {
			m.Warn().Msgf("failover.ValueToState fail:%s\n", err)
			continue
		}

		// restartRunning也会修复死任务，当restartRunning和failover并发访问时，需要NeedFix去重
		if defaultStore.NeedFix(m.ctx, state) {
			defaultStore.AssignMutex(m.ctx, model.KeyVal{Key: string(rsp.Kvs[0].Key), Val: string(rsp.Kvs[0].Value)}, true)
			_, err = defaultKVC.Delete(m.ctx, string(keyval.Key))
			if err != nil {
				m.Warn().Msgf("failover.delete fail:%s\n", err)
				continue
			}
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

			if defaultStore.NeedFix(m.ctx, state) {
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
