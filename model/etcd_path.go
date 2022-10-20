package model

import (
	"fmt"
	"strings"
)

var (
	//gate模块在etcd注册的node信息, 作用是
	// 在内网模式，让runtime发现gate
	// val是ip
	GateNodePrefix = "/scheduler/node/gate"

	//全局任务队列, 消费者是mjobs模块，使用一定的负载均衡策略分配任务
	GlobalTaskPrefix = "/scheduler/global/runq/task/data"

	//全局任务队列， 状态canrun/running/stop
	GlobalTaskPrefixState = "/scheduler/global/runq/task/state"

	//本地任务队列, key是LocalRuntimeTaskPrefix+runtime.name
	//如果runtime的连接的gateway进程异常退出，也不需要迁移任务
	LocalRuntimeTaskPrefix = "/scheduler/local/runq/task/runtime"

	//注册的runtime节点信息, 路径后面是runtimeName
	RuntimeNodePrefix = "/scheduler/runtime"

	//分配task用的分布式锁
	AssignTaskMutex = "/scheduler/task/assign/mutex"
)

const (
	CanRun  = "canrun"  //可以运行，任务被创建时的状态
	Running = "running" //任务被分配之后，运行中
	Stop    = "stop"    //这个任务被中止
)

// 生成本地任务队列全路径
func WatchLocalRuntimePrefix(runtimeName string) string {
	return fmt.Sprintf("%s/%s", LocalRuntimeTaskPrefix, runtimeName)
}

// 生成本地任务队列全路径
func fullLocalRuntimeTask(runtimeName, taskName string) string {
	return fmt.Sprintf("%s/%s/%s", LocalRuntimeTaskPrefix, runtimeName, taskName)
}

// runtimeNode转成本地队列
func RuntimeNodeToLocalTaskPrefix(fullRuntimeName string) string {
	runtimeName := takeNameFromPath(fullRuntimeName)
	return WatchLocalRuntimePrefix(runtimeName)
}

// runtimeNode转成本地队列
func RuntimeNodeToLocalTask(fullRuntimeName, taskName string) string {
	runtimeName := takeNameFromPath(fullRuntimeName)
	return fullLocalRuntimeTask(runtimeName, taskName)
}

// 本地队列转成全局队列
func FullLocalToGlobalTask(fullLocalName string) string {
	name := takeNameFromPath(fullLocalName)
	return FullGlobalTask(name)
}

// 全局队列相关两个辅助函数
// 生成全局任务队列的路径
func FullGlobalTask(taskName string) string {
	return fmt.Sprintf("%s/%s", GlobalTaskPrefix, taskName)
}

// 生成全局任务状态队列的路径
func FullGlobalTaskState(taskName string) string {
	return fmt.Sprintf("%s/%s", GlobalTaskPrefixState, taskName)
}

// 从状态队列提取taskName
func TaskNameFromState(fullPath string) string {
	return takeNameFromPath(fullPath)
}

// 提取taskName
func takeNameFromPath(fullPath string) string {
	pos := strings.LastIndex(fullPath, "/")
	if pos == -1 {
		return ""
	}
	return fullPath[pos+1:]
}
