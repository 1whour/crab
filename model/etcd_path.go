package model

import (
	"fmt"
	"strings"
)

var (
	//gate模块在etcd注册的node信息, 作用是
	// 1.在内网模式，让runtime发现gate
	// val是ip
	GateNodePrefix = "/scheduler/node/gate"

	//全局任务队列, 消费者是mjobs模块，使用一定的负载均衡策略分配任务
	GlobalTaskPrefix = "/scheduler/global/runq/task/data"
	//全局任务队列， 状态canrun/running/stop
	GlobalTaskPrefixState = "/scheduler/global/runq/task/state"
	//本地任务队列, key是LocalRuntimeTaskPrefix+runtime.name
	//如果runtime的连接的gateway进程异常退出，也不需要迁移任务
	LocalRuntimeTaskPrefix = "/scheduler/local/runq/task/runtime"

	//注册的runtime节点信息
	RuntineNodePrfix = "/scheduler/runtime"

	//gate绑定的runtime
	GateBindRuntime = "/scheduler/gate/bind/runtime"

	//分配task
	AssignTaskMutex = "/scheduler/task/assign/mutex"
)

const (
	CanRun  = "canrun"  //可以运行，任务被创建时的状态
	Running = "running" //任务被分配之后，运行中
	Stop    = "stop"    //这个任务被中止
)

func FullGlobalTaskPath(taskName string) string {
	return fmt.Sprintf("%s/%s", GlobalTaskPrefix, taskName)
}

func FullGlobalTaskStatePath(taskName string) string {
	return fmt.Sprintf("%s/%s", GlobalTaskPrefixState, taskName)
}

func TaskNameFromStatePath(fullPath string) string {
	pos := strings.LastIndex(fullPath, "/")
	if pos == -1 {
		return ""
	}
	return fullPath[pos:]
}
