package main

import (
	"github.com/gnh123/scheduler/gate"
	"github.com/gnh123/scheduler/rm"
	"github.com/gnh123/scheduler/runtime"
	"github.com/gnh123/scheduler/start"
	"github.com/gnh123/scheduler/stop"
	"github.com/guonaihong/clop"
)

type scheduler struct {
	// gate子命令
	gate.Gate `clop:"subcommand" usage:"The gate service is responsible for connecting and processing requests"`
	// runtime子命令
	runtime.Runtime `clop:"subcommand" usage:"Runtime is a pre compiled module that performs tasks"`
	// 从配置文件加载任务到scheduler集群中
	start.Start `clop:"subcommand" usage:"Start the current task from a configuration file"`
	// 加载配置文件中停止当前任务
	stop.Stop `clop:"subcommand" usage:"Stop current task from configuration file"`
	// 从集团中删除当前任务
	rm.Rm `clop:"subcommand" usage:"Remove current task from configuration file"`
}

func main() {
	s := scheduler{}
	clop.Bind(&s)
}
