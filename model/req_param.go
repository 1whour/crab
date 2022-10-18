package model

import "time"

type Param struct {
	//api的版本号, 如果有不兼容的修改，直接修改这个字段就行
	APIVersion string `yaml:"apiVersion" binding:"required" json:"api_version"`
	//oneRuntime, broadcast
	Kind string `yaml:"kind" json:"kind" binding:"required"`
	//create, stop, rm, update, gate会修改这个字段，传递到runtime
	Action   string        `yaml:"action" json:"action"`
	Executer ExecuterParam `json:"executor" yaml:"executor"`
}

func (p *Param) SetCreate() {
	p.Action = "create"
}

func (p *Param) SetRemove() {
	p.Action = "remove"
}

func (p *Param) SetStop() {
	p.Action = "stop"
}

func (p *Param) SetUpdate() {
	p.Action = "update"
}

func (p *Param) IsCreate() bool {
	return p.Action == "create"
}

func (p *Param) IsRemove() bool {
	return p.Action == "remove"
}

func (p *Param) IsStop() bool {
	return p.Action == "stop"
}

func (p *Param) IsUpdate() bool {
	return p.Action == "update"
}

type ExecuterParam struct {
	TaskName string `yaml:"task_name" json:"task_name" binding:"required"` //自定义执行器，需要给到TaskName
	HTTP     *HTTP  `yaml:"http" json:"http"`                              //http包
	Shell    *Shell `yaml:"shell" json:"shell"`                            //shell包
	Grpc     *Grpc  `yaml:"grpc" json:"grpc"`                              //grpc
}

// 判断是通用执行器
func (p ExecuterParam) IsGeneral() bool {
	return p.HTTP != nil || p.Shell != nil || p.Grpc != nil
}

func (p ExecuterParam) Name() string {
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

// 控制相关参数
type Control struct {
	FailureThreshold int           //任务执行失败的重试次数
	InitialDelay     time.Duration //任务被调度到执行器，等待多长时间再执行
	Timeout          time.Duration //任务执行的超时时间, 执行最长执行时间
	Period           time.Duration //任务重试时的间隔时间
}
