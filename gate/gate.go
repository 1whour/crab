package gate

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gnh123/ktuo/model"
	"github.com/gnh123/ktuo/slog"
	"github.com/gnh123/ktuo/store/etcd"
	"github.com/gnh123/ktuo/utils"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/guonaihong/gutil/jwt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{}

const (
	tokenQuery  = "token"
	tokenHeader = "X-Token"
)

// TODO, 规范下错误码

// Gate模块定位是网关
// 1.注册自己的信息至etcd中
// 2.维护runtime或者lambda过来的长连接
// 3.维护管理接口，保存到数据库中
type Gate struct {
	ServerAddr   string        `clop:"short;long" usage:"server address"`
	AutoFindAddr bool          `clop:"short;long" usage:"Automatically find unused ip:port, Only takes effect when ServerAddr is empty"`
	EtcdAddr     []string      `clop:"short;long;greedy" usage:"etcd address" valid:"required"`
	Name         string        `clop:"short;long" usage:"The name of the gate. If it is not filled, the default is uuid"`
	Level        string        `clop:"short;long" usage:"log level" default:"error"`
	LeaseTime    time.Duration `clop:"long" usage:"lease time" default:"7s"`
	WriteTime    time.Duration `clop:"long" usage:"write timeout" default:"4s"`
	DSN          string        `clop:"--dsn" usage:"database dsn" valid:"requried"`

	// etcd 租约id
	leaseID clientv3.LeaseID
	// 日志对象
	*slog.Slog
	// ctx
	ctx context.Context
	// 数据库对象
	loginDb *LoginDB
}

func (g *Gate) NodeName() string {
	return g.Name
}

var (
	defautlClient *clientv3.Client
	defaultKVC    clientv3.KV
	defaultStore  *etcd.EtcdStore
)

func (r *Gate) init() (err error) {

	r.Slog = slog.New(os.Stdout).SetLevel(r.Level).Str("gate", r.Name)
	r.getAddress()

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: r.DSN,
	}))
	if err != nil {
		return err
	}
	// 初始化数据库
	r.loginDb, err = newLoginDB(db)
	if err != nil {
		return err
	}

	r.ctx = context.TODO()
	if r.Name == "" {
		r.Name = uuid.New().String()
	}

	if r.LeaseTime < model.RuntimeKeepalive {
		r.LeaseTime = model.RuntimeKeepalive + time.Second
	}

	if defautlClient, err = utils.NewEtcdClient(r.EtcdAddr); err != nil { //初始etcd客户端
		return err
	}

	defaultKVC = clientv3.NewKV(defautlClient) // 内置自动重试的逻辑
	defaultStore, err = etcd.NewStore(r.EtcdAddr, r.Slog, nil)
	return err
}

func (r *Gate) autoNewAddr() (addr string) {

	if r.AutoFindAddr {
		r.ServerAddr = utils.GetUnusedAddr()
	}
	return r.ServerAddr
}

// 从ServerAddr获取，或者自动生成一个port
func (r *Gate) getAddress() string {
	if r.ServerAddr != "" {
		return r.ServerAddr
	}

	return r.autoNewAddr()
}

func (r *Gate) ok(c *gin.Context, msg string) {
	r.Debug().Caller(1).Msg(msg)
	c.JSON(200, gin.H{"code": 0, "message": ""})
}

func (r *Gate) error2(c *gin.Context, code int, format string, a ...any) {

	msg := fmt.Sprintf(format, a...)
	r.Error().Caller(1).Msg(msg)
	c.JSON(200, gin.H{"code": code, "message": msg})
}

// 简单的包装函数
func (r *Gate) error(c *gin.Context, code int, format string, a ...any) {

	msg := fmt.Sprintf(format, a...)
	r.Error().Caller(1).Msg(msg)
	c.JSON(500, gin.H{"code": code, "message": msg})
}

// 把task信息保存至etcd
func (r *Gate) createTask(c *gin.Context) {
	var req model.Param
	err := c.ShouldBind(&req)
	if err != nil {
		r.error(c, 500, "createTask:%v, type:%s", err, c.ContentType())
		return
	}

	r.Debug().Msgf("start create \n")
	taskName := req.Executer.TaskName
	// 创建数据队列
	globalTaskName := model.FullGlobalTask(taskName)

	// 先get，如果有值直接返回
	rsp, err := defaultKVC.Get(r.ctx, globalTaskName, clientv3.WithKeysOnly())
	if len(rsp.Kvs) > 0 {
		r.error(c, 500, "duplicate creation:%s", globalTaskName)
		return
	}

	req.SetCreate() //设置action

	err = defaultStore.LockCreateDataAndState(r.ctx, taskName, &req)
	if err != nil {
		r.error(c, 500, err.Error())
		return
	}

	r.ok(c, "createTask Execution succeeded") //返回正确业务码
}

// 删除etcd里面task信息，也直接下发命令更新runtime里面信息
func (r *Gate) deleteTask(c *gin.Context) {
	r.updateTaskCore(c, model.Rm)
}

func (r *Gate) updateTask(c *gin.Context) {
	r.updateTaskCore(c, model.Update)
}

// 更新etcd里面的task信息，置为静止，下发命令取消正在执行中的task
func (r *Gate) stopTask(c *gin.Context) {
	r.updateTaskCore(c, model.Stop)
}

// 更新etcd里面的task信息，也下发命令更新runtime里面信息
func (r *Gate) updateTaskCore(c *gin.Context, action string) {

	var req model.Param
	err := c.ShouldBind(&req)
	if err != nil {
		r.error(c, 500, "%s:%v", action, err)
		return
	}

	// 创建全局数据队列key名
	globalTaskName := model.FullGlobalTask(req.Executer.TaskName)

	// 先get，更新时如果没有值直接返回
	rsp, err := defaultKVC.Get(r.ctx, globalTaskName, clientv3.WithKeysOnly())
	if len(rsp.Kvs) == 0 {
		r.error(c, 500, "Task is empty and cannot be %s:%s", action, globalTaskName)
		return
	}

	switch action {
	case model.Update:
		req.SetUpdate()
	case model.Stop:
		req.SetStop()
	case model.Rm:
		req.SetRemove()
	}

	err = defaultStore.LockUpdateDataAndState(r.ctx, req.Executer.TaskName, &req, rsp.Kvs[0].ModRevision, model.CanRun, action)
	if err != nil {
		r.error(c, 500, err.Error())
		return
	}

	r.ok(c, fmt.Sprintf("%s Execution succeeded", action)) //返回正确业务码
}

// 该模块入口函数
func (r *Gate) SubMain() {
	if err := r.init(); err != nil {
		r.Error().Msgf("gate init fail:%s\n", err)
		return
	}

	go func() {
		if err := r.registerGateNode(); err != nil {
			r.Error().Msgf("gate:registerGateNode fail:%s\n", err)
		}
	}()

	gin.SetMode(gin.ReleaseMode)
	g := gin.New()
	// 跨域
	config := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", tokenHeader},
		AllowCredentials: false,
		AllowAllOrigins:  true,
		MaxAge:           12 * time.Hour,
	}

	g.Use(cors.New(config))
	g.GET(model.TASK_STREAM_URL, r.stream) //流式接口，主动推送任务至runtime
	g.POST(model.TASK_CREATE_URL, r.createTask)
	g.PUT(model.TASK_UPDATE_URL, r.updateTask)
	g.DELETE(model.TASK_DELETE_URL, r.deleteTask)
	g.POST(model.TASK_STOP_URL, r.stopTask)
	g.GET(model.TASK_STATUS_URL, r.status)

	g.Use(func(ctx *gin.Context) {

		// 登录不检查token
		if ctx.Request.URL.Path == model.UI_USER_LOGIN {
			return
		}

		token := ctx.Query(tokenQuery)
		if len(token) == 0 {
			token = ctx.GetHeader(tokenHeader)
		}
		_, err := jwt.ParseToken(token, secretToken)
		if err != nil {
			ctx.Abort()
			return
		}
	})

	// 注册
	g.POST(model.UI_USER_REGISTER_URL, r.register)
	// 登录
	g.POST(model.UI_USER_LOGIN, r.login)
	// 注销
	g.POST(model.UI_USER_LOGOUT, r.logout)
	// 删除用户
	g.DELETE(model.UI_USER_DELETE_URL, r.deleteUser)
	// 更新用户
	g.PUT(model.UI_USER_UPDATE, r.updateUser)
	// 获取某个用户
	g.GET(model.UI_USER_INFO, r.getUserInfo)
	// 获取用户列表
	g.GET(model.UI_USERS_INFO_LIST, r.GetUserInfoList)

	r.Debug().Msgf("gate:serverAddr:%s\n", r.ServerAddr)
	for i := 0; i < 3; i++ {
		if err := g.Run(r.ServerAddr); err != nil {
			r.Debug().Msgf("run fail:%v\n", err)
			r.autoNewAddrAndRegister()
			r.Debug().Msgf("gate:serverAddr:%s\n", r.ServerAddr)
			time.Sleep(time.Millisecond * 500)
		}
	}
}
