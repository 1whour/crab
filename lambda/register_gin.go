package lambda

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gnh123/scheduler/model"
	"github.com/gnh123/scheduler/slog"
	"github.com/gnh123/scheduler/utils"
	"golang.org/x/net/context"
)

const (
	ecodeRun      = 1 //任务运行出错
	ecodeCancel   = 2 //任务取消出错
	ecodeNotFound = 3 //本执行器里面没有这个taskName
)

// 客户端
type Client struct {
	options
	call map[string]callInfo
	sync.RWMutex
}

// call元数据
type callInfo struct {
	handler Handler
	ctx     context.Context
	cancel  context.CancelFunc
	state   State
}

// 新建基于gin的执行器
func NewGinExecutor(opts ...Option) (Executor, error) {
	c := &Client{}
	for _, o := range opts {
		o(&c.options)
	}

	// 如果slog没有设置
	if c.Slog == nil {
		c.Slog = slog.New(os.Stdout).SetLevel("debug")
	}

	if c.IP == "" {
		ips, err := utils.GetIpList()
		if err != nil {
			return nil, err
		}
		c.IP = ips[0]
	}

	return c, nil
}

// 把taskName 注册到map里面
func (c *Client) Register(taskName string, handler Handler) {
	c.Lock()
	defer c.Unlock()

	// 初始化的时候注册，为防止重复注册比如取重名，这里直接panic
	if _, ok := c.call[taskName]; ok {
		panic("task name:" + taskName + ":重复注册")
	}

	ctx, cancel := context.WithCancel(context.TODO())
	c.call[taskName] = callInfo{handler: handler, ctx: ctx, cancel: cancel}
}

// 写错误
func (c *Client) httpWriteErr(hcode int, ctx *gin.Context, code int, msg string) {
	ctx.JSON(hcode, gin.H{"code": code, "message": msg})
	c.Error().Msgf("code:%d, msg:%s\n", code, msg)
}

// 运行handler的函数
func (c *Client) run(ctx *gin.Context) {
	var req model.Param

	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.httpWriteErr(http.StatusInternalServerError, ctx, ecodeRun, err.Error())
		return
	}

	c.Lock()
	call, ok := c.call[req.Executer.TaskName]
	if !ok { // 不存在
		c.httpWriteErr(http.StatusNotFound, ctx, ecodeNotFound, fmt.Sprintf("找不到task:%s", req.Executer.TaskName))
		c.Unlock()
		return
	}

	// 从http net/http继承ctx, 如果tcp被close, 这个ctx就会被触发
	hctx := ctx.Request.Context()
	select {
	case <-call.ctx.Done():
		call.ctx, call.cancel = context.WithCancel(hctx)
		call.state = Running
	default:
	}

	c.Unlock()

	// 执行handler
	call.handler(ctx)

	c.Lock()

	call.state = Unused
	c.call[req.Executer.TaskName] = call

	call.cancel() //排掉
	c.Unlock()
}

// 取消现在运行中的函数
func (c *Client) cancel(ctx *gin.Context) {
	var req model.Param

	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.httpWriteErr(http.StatusInternalServerError, ctx, ecodeCancel, err.Error())
		return
	}

	c.Lock()
	defer c.Unlock()
	call := c.call[req.Executer.TaskName]

	// 利用cancel取消正在运行中的task
	if call.state == Running {
		call.cancel()
	}
}

// 如果是改造现有的gin http回调函数让其支持scheduler调度，在r.Run()接口之前调用
func (c *Client) OnlyRegisterURL(g *gin.Engine) {
	g.POST(model.LAMBDA_RUN_CANCEL, c.cancel)
	g.POST(model.LAMBDA_RUN_URL, c.run)
}

// 第一个参数是gin引擎，第二个可选参数是地址
// c.RegisterAndRun(g)
// c.RegisterAndRun(g, ":1234")
func (c *Client) RegisterAndRun(g *gin.Engine, a ...string) {
	c.OnlyRegisterURL(g)

	addr := ""
	if len(a) > 0 {
		addr = a[0]
	}

	g.Run(addr)
}
