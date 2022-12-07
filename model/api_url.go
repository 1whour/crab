package model

const (
	// 管理task相关接口
	TASK_STREAM_URL = "/ktuo/task/stream"
	TASK_CREATE_URL = "/ktuo/task/"
	TASK_DELETE_URL = "/ktuo/task/"
	TASK_UPDATE_URL = "/ktuo/task/"
	TASK_STOP_URL   = "/ktuo/task/stop"
	TASK_STATUS_URL = "/ktuo/status/"

	// user 管理相关接口
	// 注册新用户, POST
	UI_USERS_REGISTER_URL = "/ktuo/ui/users/register"
	// 用户登录, POST
	UI_USERS_LOGIN = "/ktuo/ui/users/login"
	// 删除用户, DELETE
	UI_USERS_DELETE_URL = "/ktuo/ui/users/:id"
	// 获取一批用户信息或者单个，如果带name就过滤单个用户的信息
	UI_USERS_INFO = "/ktuo/ui/users"
)
