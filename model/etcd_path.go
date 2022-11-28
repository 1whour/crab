package model

import (
	"fmt"
	"strings"
)

const (
	//gate模块在etcd注册的node信息, 作用是
	// 在内网模式，让runtime发现gate
	// val是ip
	GateNodePrefix = "/scheduler/v1/node/gate"

	//全局任务队列, 消费者是mjobs模块，使用一定的负载均衡策略分配任务
	GlobalTaskPrefix = "/scheduler/v1/global/runq/task/data"

	//全局任务队列， 状态canrun/running/stop
	GlobalTaskPrefixState = "/scheduler/v1/global/runq/task/state"

	//本地任务队列, key是LocalRuntimeTaskPrefix+runtime.name
	//如果runtime的连接的gateway进程异常退出，也不需要迁移任务
	LocalRuntimeTaskPrefix = "/scheduler/v1/local/runq/task/runtime"

	//注册的runtime节点信息, 路径后面是runtimeName
	RuntimeNodePrefix = "/scheduler/v1/runtime"

	LambdaKey = "lambda"

	//注册的runtime节点信息, 路径后面是runtimeName
	RuntimeNodeLambdaPrefix = "/scheduler/v1/runtime/lambda"

	//分配task用的分布式锁
	AssignTaskMutexPrefix = "/scheduler/v1/task/assign/mutex"
)

// 加锁需调用该函数，生成唯一的锁key
func AssignTaskMutex(fullPath string) string {
	taskName := TaskName(fullPath)
	return fmt.Sprintf("%s/%s", AssignTaskMutexPrefix, taskName)
}

// 生成本地任务队列全路径
func WatchLocalRuntimePrefix(runtimeName string) string {
	return fmt.Sprintf("%s/%s", LocalRuntimeTaskPrefix, runtimeName)
}

// runtimeNode转成本地队列前缀 路径
func ToLocalTaskPrefix(fullRuntimeName string) string {
	runtimeName := takeNameFromPath(fullRuntimeName)
	return WatchLocalRuntimePrefix(runtimeName)
}

// runtimeNode转成本地队列
func ToLocalTask(fullRuntimeName, taskName string) string {
	runtimeName := takeNameFromPath(fullRuntimeName)
	return fmt.Sprintf("%s/%s/%s", LocalRuntimeTaskPrefix, runtimeName, taskName)

}

// 转成全局队列
func ToGlobalTask(fullLocalName string) string {
	name := takeNameFromPath(fullLocalName)
	return FullGlobalTask(name)
}

// 转成全局状态队列的key
func ToGlobalTaskState(fullOrTaskName string) string {
	taskName := takeNameFromPath(fullOrTaskName)
	return FullGlobalTaskState(taskName)
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

// 从路径提取taskName
func TaskName(fullPath string) string {
	return takeNameFromPath(fullPath)
}

// 提取taskName
func takeNameFromPath(fullPath string) string {
	pos := strings.LastIndex(fullPath, "/")
	if pos == -1 {
		return fullPath
	}
	return fullPath[pos+1:]
}
