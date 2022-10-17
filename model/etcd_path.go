package model

var (
	//gate模块在etcd里面的path前缀, 1.内网模式，runtime发现gate 2.mjos分发任务到gate
	GateNodePrefix = "/scheduler/node/gate"
	//该runtime的所有任务, 在任务失败时会使用这个表的信息
	RuntimeTaskPrefix = "/scheduler/task/runtime/"
	//后面跟taskName, 通常由uuid组成, 所以任务的列表, 有几个状态, 静止，运行，待分配
	AllTaskPrefix = "/scheduler/task/all"
	//value值就是task id, 通过心跳检查runtime是否活跃，如果runtime挂掉，mjobs分启动重新分配
	//为了防止雪崩，mjobs在分配失败会标记任务状态为待分配
	RuntimeActivity = "/scheduler/runtime/activity" //value是runtime的uuid
)
