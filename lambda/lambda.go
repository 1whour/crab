package lambda

import (
	"os"
	"reflect"
	"runtime"
	"sync"
	"time"

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
	call map[string]callInfo
	sync.RWMutex
	sync.Once
	GateAddr     string `clop:"short;long" usage:"gate addr"`
	WriteTimeout time.Duration
	mu           sync.Mutex
}

// call元数据
type callInfo struct {
	ctx     context.Context
	cancel  context.CancelFunc
	handler func()
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

// 运行handler的函数
func (c *Lambda) run() {
	var req model.Param

	c.Lock()
	call, ok := c.call[req.Executer.TaskName]
	if !ok { // 不存在
		c.Unlock()
		return
	}

	c.Unlock()

	// 执行handler
	//call.handler(context.TODO())

	c.Lock()

	call.state = Unused
	c.call[req.Executer.TaskName] = call

	call.cancel() //排掉
	c.Unlock()
}

// 取消现在运行中的函数
func (c *Lambda) cancel() {
	var req model.Param

	c.Lock()
	defer c.Unlock()
	call := c.call[req.Executer.TaskName]

	// 利用cancel取消正在运行中的task
	if call.state == Running {
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
func (l *Lambda) start(handler any, funcName string) {

	l.Lock()
	defer l.Unlock()

	// 初始化的时候注册，为防止重复注册比如取重名，这里直接panic
	if _, ok := l.call[funcName]; ok {
		panic("task name:" + funcName + ":重复注册")
	}

	//c.call[funcName] =
	//ctx, cancel := context.WithCancel(context.TODO())
	//c.call[funcName] = callInfo{handler: handler, ctx: ctx, cancel: cancel}
}

func (l *Lambda) StartWithName(handler any, funcName string) {
	l.start(handler, funcName)
}

func (l *Lambda) executer(conn *websocket.Conn, param *model.Param) error {
	return nil
}

func (l *Lambda) loop() {
	gs := gatesock.New(l.Slog, l.executer, l.GateAddr, l.RuntimeName, l.WriteTimeout, &l.mu)
	gs.CreateConntion()
}

func (l *Lambda) Start(handler any) {
	l.StartWithName(handler, getFuncName(handler))
}

func (l *Lambda) Run() {
	l.loop()
}
