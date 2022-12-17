package lambda

import (
	"os"
	"reflect"
	"runtime"

	"github.com/antlabs/gstl/rwmap"
	"github.com/1whour/ktuo/executer"
	"github.com/1whour/ktuo/model"
	myruntime "github.com/1whour/ktuo/runtime"
	"github.com/1whour/ktuo/slog"
	"golang.org/x/net/context"
)

// 客户端
type Lambda struct {
	Namespace string `json:"namespace"` // TODO
	GroupId   string `json:"groupID"`   // TODO
	call      rwmap.RWMap[string, callInfo]
	myruntime.Runtime
}

// call元数据
type callInfo struct {
	ctx     context.Context
	cancel  context.CancelFunc
	handler Handler
}

// 新建基于gin的执行器
func New(opts ...Option) (*Lambda, error) {
	l := &Lambda{}
	for _, o := range opts {
		o(l)
	}

	// 如果slog没有设置
	if l.Slog == nil {
		l.Slog = slog.New(os.Stdout).SetLevel("debug")
	}

	// 给个默认超时时间
	if l.WriteTimeout == 0 {
		l.WriteTimeout = model.RuntimeKeepalive
	}

	return l, nil
}

// 获取函数的名字, 会带包名一起打印出来
func getFuncName(x any) string {
	return runtime.FuncForPC(reflect.ValueOf(x).Pointer()).Name()
}

//	func ()
//	func () error
//	func (TIn) error
//	func () (TOut, error)
//	func (TIn) (TOut, error)
//	func (context.Context) error
//	func (context.Context, TIn) error
//	func (context.Context) (TOut, error)
//	func (context.Context, TIn) (TOut, error)
//
// Where "TIn" and "TOut" are types compatible with the "encoding/json" standard library.
func (l *Lambda) start(handler any, funcName string) error {

	h, err := reflectHandler(handler)
	if err != nil {
		return err
	}

	_, ok := l.call.LoadOrStore(funcName, callInfo{handler: h})
	if ok {
		// 初始化的时候注册，为防止重复注册比如取重名，这里直接panic
		panic("task name:" + funcName + ":重复注册")
	}

	return nil
}

// 流入业务函数，自定义名字
func (l *Lambda) StartWithName(handler any, funcName string) error {
	return l.start(handler, funcName)
}

// 注入业务函数, 函数的名字就是包名.函数名 比如main.Hello
func (l *Lambda) Start(handler any) error {
	return l.StartWithName(handler, getFuncName(handler))
}

// 运行
func (l *Lambda) Run() error {
	/*
		  gs := gatesock.New(r.Slog, r.runCrudCmd, addr, r.NodeName, r.WriteTimeout, &r.MuConn, false)
			gs := gatesock.New(l.Slog, l.executer, l.Endpoint[0], l.NodeName, l.WriteTimeout, &l.MuConn, true)
			return gs.CreateConntion()
	*/
	return l.Runtime.Run(true, func() {

		executer.Register("lambda"+l.NodeName, l.createLambdaExecuter)
	})
}
