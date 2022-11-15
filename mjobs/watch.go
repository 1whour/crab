package mjobs

import (
	"github.com/gnh123/scheduler/model"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// watch 全局任务队列的变化
func (m *Mjobs) watchGlobalTaskState() {
	// 先获取节点状态
	rsp, err := defaultKVC.Get(m.ctx, model.GlobalTaskPrefixState, clientv3.WithPrefix(), clientv3.WithFilterDelete())
	if err == nil {
		for _, ev := range rsp.Kvs {
			key := string(ev.Key)
			value := string(ev.Value)

			state, err := model.ValueToState(ev.Value)
			if err != nil {
				m.Error().Msgf("watchGlobalTaskState, value to state %s\n", err)
				continue
			}

			if state.IsCanRun() {
				go m.assignMutex(newKv(key, value, int(rsp.Header.Revision)), false)
			}
		}
	}

	rev := rsp.Header.Revision + 1
	readGlobal := defautlClient.Watch(m.ctx, model.GlobalTaskPrefixState, clientv3.WithPrefix(), clientv3.WithRev(rev))
	for ersp := range readGlobal {
		for _, ev := range ersp.Events {
			key := string(ev.Kv.Key)
			value := string(ev.Kv.Value)

			m.Debug().Msgf("watch create(%t) update(%t) delete(%t) global task:%s, state:%s, version:%d\n",
				ev.IsCreate(), ev.IsModify(), ev.Type == clientv3.EventTypeDelete,
				ev.Kv.Key, ev.Kv.Value,
				ev.Kv.Version)

			switch {
			case ev.IsCreate(), ev.IsModify():

				go m.assignMutex(newKv(key, value, int(ev.Kv.ModRevision)), false)
			case ev.Type == clientv3.EventTypeDelete:
			}
		}
	}
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
				// 有新的runtimeNode节点加入
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
