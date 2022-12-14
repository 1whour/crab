package executer

import (
	"context"
	"errors"
	"sync"

	"github.com/1whour/crab/model"
)

type Executer interface {
	Run() ([]byte, error) //运行
	Stop() error          //取消
}

// key是执行器的名字，value是执行器的构造函数
var executerPlugin sync.Map

type createHandler func(ctx context.Context, param *model.Param) Executer

func Register(name string, create createHandler) {
	_, ok := executerPlugin.LoadOrStore(name, create)
	if ok {
		panic("已存在:" + name)
	}
}

func CreateExecuter(ctx context.Context, param *model.Param) (e Executer, err error) {
	e2, ok := executerPlugin.Load(param.Executer.Name())
	if !ok {
		return nil, errors.New("not found:" + param.Executer.Name())
	}

	return e2.(createHandler)(ctx, param), nil
}
