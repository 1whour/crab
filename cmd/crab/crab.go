package main

import (
	"github.com/1whour/crab/cmd/clicrud"
	"github.com/1whour/crab/cmd/etcd"
	"github.com/1whour/crab/cmd/mocksrv"
	"github.com/1whour/crab/cmd/status"
	"github.com/1whour/crab/gate"
	"github.com/1whour/crab/mjobs"
	"github.com/1whour/crab/monomer"
	"github.com/1whour/crab/runtime"
	"github.com/guonaihong/clop"
)

type crab struct {
	// gate子命令
	gate.Gate `clop:"subcommand" usage:"The gate service is responsible for connecting and processing requests"`
	// runtime子命令
	runtime.Runtime `clop:"subcommand" usage:"Runtime is a pre compiled module that performs tasks"`
	// 从配置文件加载任务到crab集群中
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
	// 单体模式，相当于起了一个runtime, gate, mjobs
	monomer.Monomer `clop:"subcommand" usage:"monomer"`
}

func main() {
	s := crab{}
	clop.Bind(&s)
}
