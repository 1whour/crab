package model

const (
	// 运行的地址
	LAMBDA_RUN_URL    = "/_github.com/gnh123/scheduler/run"
	LAMBDA_RUN_CANCEL = "/_github.com/gnh123/scheduler/cancel"

	// 管理task相关接口
	TASK_STREAM_URL = "/scheduler/task/stream"
	TASK_CREATE_URL = "/scheduler/task/"
	TASK_DELETE_URL = "/scheduler/task/"
	TASK_UPDATE_URL = "/scheduler/task/"
	TASK_STOP_URL   = "/scheduler/task/stop"
	TASK_STATUS_URL = "/scheduler/status/"
)
