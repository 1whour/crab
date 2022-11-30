package etcd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gnh123/ktuo/model"
	"github.com/gnh123/ktuo/utils"
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

		// TODO, 换成支持前缀查找的数据结构
		// 先只支持一个节点
		prefix := model.ToLocalTaskLambdaPrefix(state.TaskName)
		_, ok := e.LambdaNode.Load(prefix)
		if ok {
			return prefix, nil
		}

		e.Debug().Msgf("lambda not found:prefix(%s): taskName(%s)", prefix, state.TaskName)
		return "", fmt.Errorf("lambda not found:%s", prefix)
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

	// 获取状态数据
	rspState, err := e.defaultKVC.Get(ctx, oneTask.Key)
	if err != nil {
		e.Warn().Msgf("failover:(%t) ", failover)
		return err
	}
	// 解析成结构体
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
	if !failover && !state.IsCreate() {
		e.Debug().Msgf("state:%v\n", state)
		runtimeNode = state.RuntimeNode
	}

	e.Debug().Msgf("assign, taskName %s, action:(%s)\n", taskName, oneTask.State.Action)
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
