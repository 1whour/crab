package gatesock

import (
	"strings"
	"sync"
	"time"

	"github.com/1whour/crab/model"
	"github.com/1whour/crab/slog"
	"github.com/1whour/crab/utils"
	"github.com/gorilla/websocket"
)

type Callback func(conn *websocket.Conn, param *model.Param) (payload []byte, err error)

type GateSock struct {
	*slog.Slog
	callback     Callback
	gateAddr     string
	name         string
	id           string
	writeTimeout time.Duration
	lambda       bool
	mu           *sync.Mutex
}

func New(slog *slog.Slog, cb Callback, gateAddr string, name string, writeTimeout time.Duration, mu *sync.Mutex, lambda bool, id string) *GateSock {
	return &GateSock{Slog: slog, callback: cb, gateAddr: gateAddr, name: name, writeTimeout: writeTimeout, mu: mu, lambda: lambda, id: id}
}

// 接受来自gate服务的命令, 执行并返回结果
func (g *GateSock) readLoop(conn *websocket.Conn) error {

	go func() {
		// 对conn执行心跳检查，conn可能长时间空闲，为是检查conn是否健康，加上心跳
		for {
			time.Sleep(model.RuntimeKeepalive)
			if err := g.writeWhoami(conn); err != nil {
				g.Warn().Msgf("write whoami:%s\n", err)
				conn.Close() //关闭conn. ReadJOSN也会出错返回
				return
			}
		}
	}()

	for {
		var param model.Param
		err := conn.ReadJSON(&param) //这里不加超时时间, 一直监听gate推过来的信息
		if err != nil {
			return err
		}

		go func() {
			g.Debug().Msgf("crud action:%s, taskName:%s\n", param.Action, param.Executer.TaskName)
			payload, err := g.callback(conn, &param)
			if err != nil {
				g.Error().Msgf("runtime.runCrud, action(%s):%s\n", param.Action, err)
				//r.writeError(conn, r.WriteTimeout, 1, err.Error())
				return
			}
			// TODO, 把结果回写入mysql中
			_ = payload
		}()
	}

}

func genGateAddr(gateAddr string) string {
	if strings.HasPrefix(gateAddr, "ws://") || strings.HasPrefix(gateAddr, "wss://") {
		return gateAddr
	}
	return "ws://" + gateAddr
}

func (g *GateSock) writeWhoami(conn *websocket.Conn) (err error) {
	g.mu.Lock()
	err = utils.WriteJsonTimeout(conn, model.Whoami{Name: g.name, Lambda: g.lambda, Id: g.id}, g.writeTimeout)
	g.mu.Unlock()
	return err
}

// 创建一个长连接
func (g *GateSock) CreateConntion() error {

	gateAddr := genGateAddr(g.gateAddr) + model.TASK_STREAM_URL
	c, _, err := websocket.DefaultDialer.Dial(gateAddr, nil)
	if err != nil {
		g.Error().Msgf("runtime:dial:%s, address:%s\n", err, gateAddr)
		return err
	}

	defer c.Close()

	if err := g.writeWhoami(c); err != nil {
		return err
	}

	return g.readLoop(c)
}
