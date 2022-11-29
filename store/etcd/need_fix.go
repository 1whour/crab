package etcd

import (
	"context"
	"time"

	"github.com/gnh123/scheduler/model"
	clientv3 "go.etcd.io/etcd/client/v3"
)

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
func (m *EtcdStore) NeedFix(ctx context.Context, state model.State) bool {
	if state.IsFailed() {
		return true
	}

	var ip *clientv3.GetResponse
	var err error
	// runtimeNode为空
	if len(state.RuntimeNode) == 0 {
		goto next
	}

	ip, err = m.defaultKVC.Get(ctx, state.RuntimeNode)
	if err != nil {
		//m.Error().Msgf("restartRunning: 2 get ip %v, key(%s)\n", err, state.RuntimeNode)
		goto next
	}

	// 这里加上UpdateTime的原因:
	// gate: watch任务， write socket , ack状态回写etcd
	//                   mjobs NeedFix()
	// 就会发现NeedFix() 会导致这个任务被重放了一遍
	if (state.IsStop() || state.IsRemove()) && state.InRuntime && len(ip.Kvs) > 0 && time.Since(state.UpdateTime) > model.RuntimeKeepalive+1 {
		//m.Error().Msgf("restartRunning: 3")
		return true
	}

next:
	return (state.IsCreate() || state.IsUpdate()) && (ip == nil || len(ip.Kvs) == 0)
}
