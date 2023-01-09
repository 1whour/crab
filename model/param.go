package model

import (
	"time"
)

// 给delete, stop, continue使用
type OnlyParam struct {
	Action   string `yaml:"action" json:"action"`
	Executer struct {
		TaskName string `yaml:"taskName" json:"taskName" binding:"required"` //自定义执行器，需要给到TaskName
	} `json:"executer" yaml:"executer"`
}

type Param struct {
	//api的版本号, 如果有不兼容的修改，直接修改这个字段就行
	APIVersion string  `yaml:"apiVersion" binding:"required" json:"apiVersion"`
	Trigger    Trigger `yaml:"trigger" binding:"required" json:"trigger"`
	//oneRuntime, broadcast
	Kind string `yaml:"kind" json:"kind" binding:"required"`
	//create, stop, rm, update, gate会修改这个字段，方便传递到runtime
	Action   string        `yaml:"action" json:"action"`
	Executer ExecuterParam `json:"executer" yaml:"executer"`
	//ExecTime time.Time     `json:"execTime" yaml:"execTime"`
}

const (
	Create   = "create"
	Rm       = "remove"
	Stop     = "stop"
	Update   = "update"
	Continue = "continue"
)

// 任务的触发器
type Trigger struct {
	Cron string `yaml:"cron" json:"cron" binding:"required"`
	Once string `yaml:"once" json:"once"`
}

func (p *Param) IsLambda() bool {
	return p.Executer.Lambda != nil && p.Executer.Lambda.Funcs != nil
}

func (p *Param) SetCreate() {
	p.Action = Create
}

func (p *Param) SetRemove() {
	p.Action = Rm
}

func (p *Param) SetStop() {
	p.Action = Stop
}

func (p *Param) SetUpdate() {
	p.Action = Update
}

func (p *Param) IsCreate() bool {
	return p.Action == Create
}

func (p *Param) IsRemove() bool {
	return p.Action == Rm
}

func (p *Param) IsStop() bool {
	return p.Action == Stop
}

func (p *Param) IsUpdate() bool {
	return p.Action == Update
}

func (p *Param) IsContinue() bool {
	return p.Action == Continue
}

type ExecuterParam struct {
	TaskName  string  `yaml:"taskName" json:"taskName" binding:"required"` //自定义执行器，需要给到TaskName
	GroupName string  `yaml:"groupName" json:"groupName"`                  //TODO, 还没想好
	HTTP      *HTTP   `yaml:"http" json:"http"`                            //http包
	Shell     *Shell  `yaml:"shell" json:"shell"`                          //shell包
	Grpc      *Grpc   `yaml:"grpc" json:"grpc"`                            //grpc
	Lambda    *Lambda `yaml:"lambda" json:"lambda"`                        //lambda
}

// 判断是通用执行器
func (p ExecuterParam) IsGeneral() bool {
	return p.HTTP != nil || p.Shell != nil || p.Grpc != nil
}

// name
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

	if p.Lambda != nil && p.Lambda.Funcs != nil {
		return "lambda" + p.TaskName
	}
	return ""
}

// 控制相关参数
type Control struct {
	FailureThreshold int           //任务执行失败的重试次数
	InitialDelay     time.Duration //任务被调度到执行器，等待多长时间再执行
	Timeout          time.Duration //任务执行的超时时间, 执行最长执行时间
	Period           time.Duration //任务重试时的间隔时间
}
