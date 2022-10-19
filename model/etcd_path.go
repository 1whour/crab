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

	//runtime绑定到gate
	RuntimeBindGate = "/scheduler/runtime/bind/gate"

	//分配task用的分布式锁
	AssignTaskMutex = "/scheduler/task/assign/mutex"
)

const (
	CanRun  = "canrun"  //可以运行，任务被创建时的状态
	Running = "running" //任务被分配之后，运行中
	Stop    = "stop"    //这个任务被中止
)

// 生成runtime节点的信息
func FullRuntimeNodePath(runtimeName string) string {
	return fmt.Sprintf("%s/%s", RuntimeNodePrefix, runtimeName)
}

// 生成本地任务队列全路径
func LocalRuntimeTaskPath(runtimeName, taskName string) string {
	return fmt.Sprintf("%s/%s/%s", LocalRuntimeTaskPrefix, runtimeName, taskName)
}

// runtimeNode转成本地队列
func RuntimeNodeToLocalTaskPath(fullRuntimeName, taskName string) string {
	name := takeNameFromPath(fullRuntimeName)
	return LocalRuntimeTaskPath(name, taskName)
}

// 生成全局任务队列的路径
func FullGlobalTaskPath(taskName string) string {
	return fmt.Sprintf("%s/%s", GlobalTaskPrefix, taskName)
}

// 生成全局任务状态队列的路径
func FullGlobalTaskStatePath(taskName string) string {
	return fmt.Sprintf("%s/%s", GlobalTaskPrefixState, taskName)
}

// gate node的信息
func FullGateNodePath(name string) string {
	return fmt.Sprintf("%s/%s", GateNodePrefix, name)
}

// 从状态队列提取taskName
func TaskNameFromStatePath(fullPath string) string {
	return takeNameFromPath(fullPath)
}

func takeNameFromPath(fullPath string) string {
	pos := strings.LastIndex(fullPath, "/")
	if pos == -1 {
		return ""
	}
	return fullPath[pos:]
}
