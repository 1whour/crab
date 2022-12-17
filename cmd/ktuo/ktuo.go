package main

import (
	"github.com/1whour/ktuo/cmd/clicrud"
	"github.com/1whour/ktuo/cmd/etcd"
	"github.com/1whour/ktuo/cmd/mocksrv"
	"github.com/1whour/ktuo/cmd/status"
	"github.com/1whour/ktuo/gate"
	"github.com/1whour/ktuo/mjobs"
	"github.com/1whour/ktuo/runtime"
	"github.com/guonaihong/clop"
)

type ktuo struct {
	// gate子命令
	gate.Gate `clop:"subcommand" usage:"The gate service is responsible for connecting and processing requests"`
	// runtime子命令
	runtime.Runtime `clop:"subcommand" usage:"Runtime is a pre compiled module that performs tasks"`
	// 从配置文件加载任务到ktuo集群中
	clicrud.Start `clop:"subcommand" usage:"Start the current task from a configuration file"`
	// 加载配置文件中停止当前任务
	clicrud.Stop `clop:"subcommand" usage:"Stop current task from configuration file"`
	// 从集群中删除当前任务
	clicrud.Rm `clop:"subcommand" usage:"Remove current task from configuration file"`
	// 使用新配置替换集群中的配置文件
	clicrud.Update `clop:"subcommand" usage:"Update current task from configuration file"`
	// mjobs子命令，负责任务分发，故障转移
	mjobs.Mjobs `clop:"subcommand" usage:"mjobs"`
	// etcd子命令, 主要用于自测
	etcd.Etcd `clop:"subcommand" usage:"etcd"`
	// mocksrv子命令， 主要用于自测
	mocksrv.MockSrv `clop:"subcommand" usage:"mock server"`
	// 查看任务状态
	status.Status `clop:"subcommand" usage:"status"`
}

func main() {
	s := ktuo{}
	clop.Bind(&s)
}
