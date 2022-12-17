package mocksrv

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/1whour/ktuo/model"
)

// 本子命令主要是为了测试脚本而写
// 保存所有的任务
type MockSrv struct {
	Addr string `clop:"short;long" usage:"server address" default:":8181"`
	m    map[string]int
	mu   sync.RWMutex
}

func (m *MockSrv) handler(g *gin.Engine) {
	g.POST("/task", func(c *gin.Context) {
		key := c.GetHeader(model.DefaultExecuterHTTPKey)
		m.mu.Lock()
		m.m[key]++
		m.mu.Unlock()
		c.String(200, "")
	})

	g.GET("/task", func(c *gin.Context) {
		key := c.GetHeader(model.DefaultExecuterHTTPKey)
		m.mu.Lock()
		val := m.m[key]
		m.mu.Unlock()
		c.String(200, fmt.Sprint(val))
	})
}

// 子命令入口函数
func (m *MockSrv) SubMain() {
	m.m = make(map[string]int)
	r := gin.Default()
	m.handler(r)
	r.Run(m.Addr)
}
