package etcd

import (
	"context"
	"encoding/json"
	"time"

	"github.com/1whour/crab/model"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// 异常条件:
// 1.mjobs模板把任务分配到本地队列状态是running, 如果这时候runtime挂了, runtimeNode会被从etcd自动删除

// 2.如果把任务写回至runtime websocket连接挂了，runtime进程也死了，可以走进第一个逻辑恢复, 1,2 算一种异常

// 3.如果gate把任务分配至runtime这时候连接挂了，runtime进程还在，需要把任务的状态修改(gate)为failed, 下一次重新分配

// 4.如果任务是Stop, Rm的任务仅仅保证执行一次

// 5.如果相同taskName的runtime被重启了。

// 做法:
// 集群稳定的前提下(当runtime的个数>=1 gate的个数>=1)，什么样的任务可以被恢复?

// 1.如果是Create和Update的任务，任务绑定的runtime是空, State是任何状态，都需要被恢复, 这是一个还需要被运行的状态

// 2.如果是Stop和Rm的任务, 如果runtimeNode不为空。Number == 0时会尝试一次

// 5.新加id字段，必须保证唯一性，taskName相同，id不同，说明被重启了，需要同步数据
func (m *EtcdStore) NeedFix(ctx context.Context, state model.State) bool {
	if state.IsFailed() {
		return true
	}

	var runtimeInfo *clientv3.GetResponse
	var err error
	var info model.RegisterRuntime
	// runtimeNode为空
	if len(state.RuntimeNode) == 0 {
		goto next
	}

	runtimeInfo, err = m.defaultKVC.Get(ctx, state.RuntimeNode)
	if err != nil {
		//m.Error().Msgf("restartRunning: 2 get ip %v, key(%s)\n", err, state.RuntimeNode)
		goto next
	}

	if len(runtimeInfo.Kvs) > 0 {

		if err = json.Unmarshal(runtimeInfo.Kvs[0].Value, &info); err != nil {
			m.Warn().Msgf("Unmarshal runtiemInfo fail:%s", err)
		}
	}
	// 这里加上UpdateTime的原因:
	// gate: watch任务， write socket , ack状态回写etcd
	//                   mjobs NeedFix()
	// 一个go程已恢复这个任务正在写socket，同时另一个go程刚好触发NeedFix()逻辑 会导致这个任务被重放了一遍
	if (state.IsStop() || state.IsRemove()) && state.InRuntime && len(runtimeInfo.Kvs) > 0 && time.Since(state.UpdateTime) > model.RuntimeKeepalive+1 {
		//m.Error().Msgf("restartRunning: 3")
		return true
	}

next:
	// Continue
	return (state.IsCreate() || state.IsUpdate() || state.IsContinue()) &&
		(runtimeInfo == nil || len(runtimeInfo.Kvs) == 0 || len(info.Id) > 0 && info.Id != state.RuntimeID)
}
