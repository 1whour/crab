package model

const (
	// 管理task相关接口
	TASK_STREAM_URL = "/crab/task/stream"
	TASK_CREATE_URL = "/crab/task/"
	TASK_DELETE_URL = "/crab/task/"
	TASK_UPDATE_URL = "/crab/task/"
	TASK_STOP_URL   = "/crab/task/stop"
	TASK_STATUS_URL = "/crab/status/"

	// 执行任务时的保存结果
	TASK_EXECUTER_RESULT_URL = "/crab/ui/task/result"
	// 获取任务的列表
	TASK_EXECUTER_RESULT_LIST_URL = "/crab/ui/task/result/list"
	// user 管理相关接口
	// 注册新用户, POST
	UI_USER_REGISTER_URL = "/crab/ui/user"
	// 用户登录, POST
	UI_USER_LOGIN = "/crab/ui/user/login"
	// 退出
	UI_USER_LOGOUT = "/crab/ui/user/logout"
	// 删除用户, DELETE
	UI_USER_DELETE_URL = "/crab/ui/user"
	// 获取一批用户信息或者单个，如果带name就过滤单个用户的信息
	UI_USER_INFO = "/crab/ui/user/info"

	UI_USER_UPDATE = "/crab/ui/user"
	// 获取用户列表
	UI_USERS_INFO_LIST = "/crab/ui/users/list"
)
