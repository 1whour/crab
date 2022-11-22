package mjobs

import (
	"github.com/gnh123/scheduler/model"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func (m *Mjobs) todoCallTask(key string, value []byte, version int) {
	state, err := model.ValueToState(value)
	if err != nil {
		m.Error().Msgf("watchGlobalTaskState, value to state %s\n", err)
		return
	}

	if state.Ack && !state.IsFailed() {
		return
	}

	if state.IsCanRun() {
		oneTask := model.KeyVal{
			Key:     key,
			Val:     string(value),
			State:   state,
			Version: version,
		}
		go defaultStore.AssignMutex(m.ctx, oneTask, false)
	}
}

// watch 全局任务队列的变化
func (m *Mjobs) watchGlobalTaskState() {
	// 先获取节点状态
	rsp, err := defaultKVC.Get(m.ctx, model.GlobalTaskPrefixState, clientv3.WithPrefix(), clientv3.WithFilterDelete())
	if err == nil {
		for _, ev := range rsp.Kvs {
			key := string(ev.Key)

			m.todoCallTask(key, ev.Value, int(rsp.Header.Revision))
		}
	}

	rev := rsp.Header.Revision + 1
	readGlobal := defautlClient.Watch(m.ctx, model.GlobalTaskPrefixState, clientv3.WithPrefix(), clientv3.WithRev(rev))
	for ersp := range readGlobal {
		for _, ev := range ersp.Events {
			key := string(ev.Kv.Key)
			version := ev.Kv.ModRevision

			m.Debug().RawJSON("state", ev.Kv.Value).Msgf("watch create(%t) update(%t) delete(%t) global task:%s, version:%d",
				ev.IsCreate(), ev.IsModify(), ev.Type == clientv3.EventTypeDelete,
				ev.Kv.Key, version)

			switch {
			case ev.IsCreate(), ev.IsModify():

				m.todoCallTask(key, ev.Kv.Value, int(version))
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
			m.Debug().Msgf("watch mjobs.runtimeNodes key(%s) value(%s) create(%t), update(%t), delete(%t)\n",
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
