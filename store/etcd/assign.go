package etcd

import (
	"context"
	"errors"
	"time"

	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// mjobs子命令的的入口函数
// 随机选择一个runtimeNode
func (e *EtcdStore) selectRuntimeNode(state model.State) (string, error) {

	if !state.Lambda && e.RuntimeNode.RuntimeNode.Len() == 0 {
		e.Warn().Msgf("assign.runtimeNodes.size is 0\n")
		return "", errors.New("assign.runtimeNodes.size is 0")
	}

	if state.Lambda {

		if e.LambdaNode.Len() == 0 {
			return "", errors.New("assign.lambdaNodes.size is 0")
		}

		// TODO
		e.RuntimeNode.LambdaNode.Keys()
	}

	runtimeNodes := e.RuntimeNode.RuntimeNode.Keys()

	return utils.SliceRandOne(runtimeNodes), nil
}

// 使用分布式锁
func (e *EtcdStore) AssignMutex(ctx context.Context, oneTask model.KeyVal, failover bool) {
	e.AssignMutexWithCb(ctx, oneTask, failover, nil)
}

func (e *EtcdStore) LockUnlock(ctx context.Context, key string, cb func() error) error {

	mutexName := model.AssignTaskMutex(key)

	s, err := concurrency.NewSession(e.defaultClient)
	if err != nil {
		return err
	}
	defer s.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	// 创建分布式锁
	l := concurrency.NewMutex(s, mutexName)

	if err := l.Lock(ctx); err != nil {
		e.Debug().Msgf("assign lock:%s\n", err)
		return err
	}

	defer l.Unlock(ctx)

	return cb()
}

func (e *EtcdStore) AssignMutexWithCb(ctx context.Context, oneTask model.KeyVal, failover bool, cb func()) error {
	return e.LockUnlock(ctx, oneTask.Key, func() error {

		if err := e.assign(ctx, oneTask, failover); err != nil {
			e.Warn().Msgf("assign err:%v\n", err)
		}

		if cb != nil {
			cb()
		}
		return nil
	})

}

// 分配任务的逻辑
func (e *EtcdStore) assign(ctx context.Context, oneTask model.KeyVal, failover bool) error {

	rspState, err := e.defaultKVC.Get(ctx, oneTask.Key)
	if err != nil {
		e.Warn().Msgf("failover:(%t) ", failover)
		return err
	}
	state, err := model.ValueToState(rspState.Kvs[0].Value)
	if err != nil {
		e.Warn().Msgf("key: value to state:%s ", err)
		return err
	}

	// 如果这条任务需要被修复，在获取锁之后，还要重新获取最新状态
	// 很有可能这个任务已经被另一个mjobs修复过
	if failover {
		failover = e.NeedFix(ctx, state)
	}

	if !failover {
		if state.IsRunning() {
			e.Debug().Msgf("Ignore:This task has been assigned, key:%s, state:%v\n", oneTask.Key, state)
			return nil
		}
	}

	if failover {
		e.Debug().Caller(4).Msgf("failover(%t), call assign, key:%s, state:%v\n", failover, oneTask.Key, state)
	} else {
		e.Debug().Msgf("failover(%t) call assign, key:%s, state:%v\n", failover, oneTask.Key, state)
	}

	kv := oneTask

	// 从状态信息里面获取tastName
	taskName := model.TaskName(kv.Key)
	if taskName == "" {
		e.Debug().Msgf("taskName is empty, %s\n", kv.Key)
		return errors.New("taskName is empty")
	}

	// TODO 如果选出的runtimeNode和出错runtimeNode一样，需要重新选择
	runtimeNode, err := e.selectRuntimeNode(state)
	if err != nil {
		return err
	}
	// 如果runtimeNode绑定好，除了出错，或者新建，会取目前绑定的runtimeNode直接使用
	if !failover && !state.IsCreate() && !state.IsFailed() {
		e.Debug().Msgf("state:%v\n", state)
		runtimeNode = state.RuntimeNode
	}

	e.Debug().Msgf("assign, taskName %s, action:(%s)\n", taskName, oneTask.State.Action)
	rsp, err := e.defaultKVC.Get(ctx, model.FullGlobalTask(taskName), clientv3.WithRev(int64(oneTask.Version)))
	if err != nil {
		e.Error().Msgf("get global task path fail:%s\n", err)
		return err
	}

	if len(rsp.Kvs) == 0 {
		e.Warn().Msgf("get %s value is nil\n", model.FullGlobalTask(taskName))
		return err
	}

	// 如果是没有在运行中的删除任务，直接清理data, state队列中的数据
	if oneTask.State.IsRemove() && !oneTask.State.InRuntime && !oneTask.State.IsFailed() {
		e.defaultKVC.Delete(ctx, model.FullGlobalTask(taskName))
		e.defaultKVC.Delete(ctx, model.FullGlobalTaskState(taskName))
		return nil
	}

	if state.IsOneRuntime() {
		if err = e.UpdateLocalAndGlobal(ctx, taskName, runtimeNode, rspState, state.Action); err != nil {
			return err
		}

	} else if state.IsBroadcast() {

		e.RuntimeNode.RuntimeNode.Range(func(key, val string) bool {
			err = e.UpdateLocalAndGlobal(ctx, taskName, runtimeNode, rspState, state.Action)
			return err == nil
		})
	} else {
		e.Warn().Msgf("Unknown kind:%s\n", state.Kind)
	}
	// 更新状态中的值
	e.Debug().Msgf("assign bye bye: oneRuntime(%t), broadcase(%t):key(%s):value(%s)\n",
		state.IsOneRuntime(), state.IsBroadcast(), model.FullGlobalTaskState(taskName), runtimeNode)
	return err
}
