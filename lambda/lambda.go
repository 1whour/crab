package lambda

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/antlabs/gstl/rwmap"
	"github.com/gnh123/scheduler/gatesock"
	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/slog"
	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
)

const (
	ecodeRun      = 1 //任务运行出错
	ecodeCancel   = 2 //任务取消出错
	ecodeNotFound = 3 //本执行器里面没有这个taskName
)

// 客户端
type Lambda struct {
	options
	call rwmap.RWMap[string, callInfo]
	sync.Once
	GateAddr     string `clop:"short;long" usage:"gate addr"`
	WriteTimeout time.Duration
	mu           sync.Mutex
}

// call元数据
type callInfo struct {
	ctx     context.Context
	cancel  context.CancelFunc
	handler Handler
	state   State
}

// 新建基于gin的执行器
func New(opts ...Option) (*Lambda, error) {
	c := &Lambda{}
	for _, o := range opts {
		o(&c.options)
	}

	// 如果slog没有设置
	if c.Slog == nil {
		c.Slog = slog.New(os.Stdout).SetLevel("debug")
	}

	return c, nil
}

// 取消现在运行中的函数
func (c *Lambda) cancel() {
	var req model.Param

	call, ok := c.call.Load(req.Executer.TaskName)

	// 利用cancel取消正在运行中的task
	if ok && call.state == Running {
		call.cancel()
	}
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

// 执行回调函数
func (l *Lambda) executer(conn *websocket.Conn, param *model.Param) (payload []byte, err error) {
	if param.Executer.TaskName != l.RuntimeName {
		return nil, fmt.Errorf("taskName:%s != l.RuntimeName:%s", param.Executer.TaskName, l.RuntimeName)
	}

	if param.Executer.Lambda == nil {
		return nil, fmt.Errorf("lambda is nil ???")
	}

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

// 注入业务函数, 函数的名字就是包名.函数名 比如main.Hello
func (l *Lambda) Start(handler any) error {
	return l.StartWithName(handler, getFuncName(handler))
}

// 运行
func (l *Lambda) Run() {
	gs := gatesock.New(l.Slog, l.executer, l.GateAddr, l.RuntimeName, l.WriteTimeout, &l.mu)
	gs.CreateConntion()
}
