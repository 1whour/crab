package main

import (
	"github.com/gnh123/scheduler/gate"
	"github.com/guonaihong/clop"
)

type scheduler struct {
	// gate子命令
	gate.Gate `clop:"subcommand=Gate" usage:"The gate service is responsible for connecting and processing requests"`
}

func main() {
	s := scheduler{}
	clop.Bind(&s)
}
