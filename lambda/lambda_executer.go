package lambda

import (
	"fmt"

	"github.com/antlabs/gstl/rwmap"
	"github.com/1whour/crab/executer"
	"github.com/1whour/crab/model"
	"github.com/1whour/crab/slog"
	"golang.org/x/net/context"
)

var _ executer.Executer = (*lambdaExecuter)(nil)

type lambdaExecuter struct {
	param    *model.Param
	NodeName string
	slog.Slog
	call *rwmap.RWMap[string, callInfo]
}

func (l *Lambda) createLambdaExecuter(ctx context.Context, param *model.Param) executer.Executer {
	return &lambdaExecuter{
		param:    param,
		NodeName: l.NodeName,
		Slog:     *l.Slog,
		call:     &l.call,
	}
}

// TODO
func (l *lambdaExecuter) Stop() error {
	return nil
}

// 执行回调函数
func (l *lambdaExecuter) Run() (payload []byte, err error) {
	param := l.param
	if param.Executer.TaskName != l.NodeName {
		return nil, fmt.Errorf("taskName:%s != l.TaskName:%s", param.Executer.TaskName, l.NodeName)
	}

	if param.Executer.Lambda == nil {
		return nil, fmt.Errorf("lambda is nil ???")
	}

	// TODO 多个函数, 结果合并起来保存到mysql里面
	for _, f := range param.Executer.Lambda.Funcs {
		call, ok := l.call.Load(f.Name)
		if !ok {
			l.Warn().Msgf("func.name:%s is not found\n", f.Name)
			continue
		}

		rsp, err := call.handler.Invoke(call.ctx, []byte(f.Args))
		if err != nil {
			l.Warn().Msgf("call handler fail:%s\n", err)
			continue
		}

		// TODO 把结果回写到服务端
		l.Debug().Msgf("lambda result:%s\n", rsp)
	}

	return nil, nil
}
