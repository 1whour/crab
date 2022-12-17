package utils

import (
	"strings"
	"sync"
	"time"

	"github.com/1whour/ktuo/model"
	"github.com/1whour/ktuo/slog"
	"github.com/gorilla/websocket"
)

type Callback func(conn *websocket.Conn, param *model.Param) error
type GateSocket struct {
	*slog.Slog
	callback     Callback
	gateAddr     string
	name         string
	writeTimeout time.Duration
	mu           *sync.Mutex
}

func NewGateSocket(slog *slog.Slog, cb Callback, gateAddr string, name string, writeTimeout time.Duration, mu *sync.Mutex) *GateSocket {
	return &GateSocket{Slog: slog, callback: cb}
}

// 接受来自gate服务的命令, 执行并返回结果
func (g *GateSocket) readLoop(conn *websocket.Conn) error {

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
			if err := g.callback(conn, &param); err != nil {
				//if err := g.runCrudCmd(conn, &param); err != nil {
				g.Error().Msgf("runtime.runCrud, action(%s):%s\n", param.Action, err)
				//r.writeError(conn, r.WriteTimeout, 1, err.Error())
				return
			}
		}()
	}

}

func genGateAddr(gateAddr string) string {
	if strings.HasPrefix(gateAddr, "ws://") || strings.HasPrefix(gateAddr, "wss://") {
		return gateAddr
	}
	return "ws://" + gateAddr
}

func (g *GateSocket) writeWhoami(conn *websocket.Conn) (err error) {
	g.mu.Lock()
	err = WriteJsonTimeout(conn, model.Whoami{Name: g.name}, g.writeTimeout)
	g.mu.Unlock()
	return err
}

// 创建一个长连接
func (g *GateSocket) createConntion() error {

	gateAddr := genGateAddr(g.gateAddr) + model.TASK_STREAM_URL
	c, _, err := websocket.DefaultDialer.Dial(gateAddr, nil)
	if err != nil {
		g.Error().Msgf("runtime:dial:%s, address:%s\n", err, gateAddr)
		return err
	}

	defer c.Close()
	err = WriteJsonTimeout(c, model.Whoami{Name: g.name}, g.writeTimeout)
	if err != nil {
		return err
	}

	return g.readLoop(c)
}
