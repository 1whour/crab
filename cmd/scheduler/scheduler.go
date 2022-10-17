package main

import (
	"github.com/gnh123/scheduler/cmd/rm"
	"github.com/gnh123/scheduler/cmd/start"
	"github.com/gnh123/scheduler/cmd/stop"
	"github.com/gnh123/scheduler/cmd/update"
	"github.com/gnh123/scheduler/gate"
	"github.com/gnh123/scheduler/mjobs"
	"github.com/gnh123/scheduler/runtime"
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
	// 从集群中删除当前任务
	rm.Rm `clop:"subcommand" usage:"Remove current task from configuration file"`
	// 使用新配置替换集群中的配置文件
	update.Update `clop:"subcommand" usage:"Update current task from configuration file"`
	// mjobs子命令，负责任务分发，故障转移
	mjobs.Mjobs `clop:"subcommand" usage:"mjobs"`
}

func main() {
	s := scheduler{}
	clop.Bind(&s)
}
