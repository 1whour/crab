package model

type Param struct {
	Proto       string //http, grpc(TODO)
	TaskName    string //TaskName
	HTTPMessage HTTP   //http包
}

// 调度器发给执行器的包
type HTTP struct {
	Query  []string //存放查询字符串
	Header []string //存放http header
	Body   string   //存放body
}
