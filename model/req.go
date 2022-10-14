package model

import "time"

type Param struct {
	Executor string //通用执行器，shell, http, grpc
	TaskName string //自定义执行器，需要给到TaskName
	HTTP     HTTP   //http包
	Shell    Shell  //shell包
}

// 判断是通用执行器
func (p Param) IsGeneral() bool {
	switch p.Executor {
	case "http", "grpc", "shell":
		return true
	}

	return false
}

// grpc 数据包
type Grpc struct {
}

// shell数据包
type Shell struct {
	Command string   //命令
	Args    []string //命令分割成数组形式
}

// 调度器发给执行器的包
type HTTP struct {
	Scheme  string //https还是http，默认是http
	Host    string //是连接的主机名，默认是调度器的ip
	Port    int    //端口
	Path    string
	Method  string      //get post delete
	Querys  []NameValue //存放查询字符串
	Headers []NameValue //存放http header
	Body    string      //存放body
}

type NameValue struct {
	Name  string
	Value string
}

// 控制相关参数
type Control struct {
	FailureThreshold int           //任务执行失败的重试次数
	InitialDelay     time.Duration //任务被调度到执行器，等待多长时间再执行
	Timeout          time.Duration //任务执行的超时时间, 执行最长执行时间
	Period           time.Duration //任务重试时的间隔时间
}
