package model

import "time"

type ExecutorParam struct {
	TaskName string `yaml:"task_name"` //自定义执行器，需要给到TaskName
	HTTP     *HTTP  `yaml:"http"`      //http包
	Shell    *Shell `yaml:"shell"`     //shell包
	Grpc     *Grpc  `yaml:"grpc"`      //grpc
}

// 判断是通用执行器
func (p ExecutorParam) IsGeneral() bool {
	return p.HTTP != nil || p.Shell != nil || p.Grpc != nil
}

func (p ExecutorParam) Name() string {
	if p.HTTP != nil {
		return "http"
	}

	if p.Shell != nil {
		return "shell"
	}

	if p.Grpc != nil {
		return "grpc"
	}

	return ""
}

// grpc 数据包, TODO
type Grpc struct {
}

// shell数据包
type Shell struct {
	Command string   //命令
	Args    []string //命令分割成数组形式
}

// 调度器发给执行器的包
type HTTP struct {
	Scheme  string      `yaml:"scheme"` //https还是http，默认是http
	Host    string      `yaml:"host"`   //是连接的主机名，默认是调度器的ip
	Port    int         `yaml:"port"`   //端口
	Path    string      `yaml:"path"`
	Method  string      `yaml:"method"`  //get post delete
	Querys  []NameValue `yaml:"querys"`  //存放查询字符串
	Headers []NameValue `yaml:"headers"` //存放http header
	Body    string      `yaml:"body"`    //存放body
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
