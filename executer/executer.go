package executer

import (
	"context"
	"errors"
	"sync"

	"github.com/gnh123/scheduler/model"
)

type Executer interface {
	Run() error    //运行
	Cancel() error //取消
}

// key是执行器的名字，value是执行器的构造函数
var executerPlugin sync.Map

type createHandler func(ctx context.Context, param *model.ExecutorParam) Executer

func Register(name string, create createHandler) {
	_, ok := executerPlugin.LoadOrStore(name, create)
	if ok {
		panic("已存在:" + name)
	}
}

func CreateExecuter(ctx context.Context, param *model.ExecutorParam) (e Executer, err error) {
	e2, ok := executerPlugin.Load(param.Name())
	if !ok {
		return nil, errors.New("not found:" + param.Name())
	}

	return e2.(createHandler)(ctx, param), nil
}
