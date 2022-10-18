package model

var (
	//gate模块在etcd注册的node信息, 作用是
	// 1.在内网模式，让runtime发现gate
	// val是ip
	GateNodePrefix = "/scheduler/node/gate"

	//全局任务队列, 消费者是mjobs模块，使用一定的负载均衡策略分配任务
	GlobalTaskPrefix = "/scheduler/global/runq/task"

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
