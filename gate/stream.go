package gate

import (
	"github.com/gin-gonic/gin"
	"github.com/gnh123/ktuo/model"
)

func (r *Gate) stream(c *gin.Context) {

	w := c.Writer
	req := c.Request

	con, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		r.Error().Msgf("upgrade:", err)
		return
	}
	defer con.Close()

	keepalive := make(chan bool)
	runtimeNode := ""
	for {
		// 读取心跳
		req := model.Whoami{}
		err := con.ReadJSON(&req)
		if err != nil {
			r.delRuntimeNode(req)
			r.Warn().Msgf("gate.stream.read:%s\n", err)
			break
		}

		if runtimeNode == "" {
			go func() {
				r.registerRuntimeWithKeepalive(req, keepalive)
			}()
			go r.watchLocalRunq(&req, con)
			runtimeNode = req.Name
		} else {
			keepalive <- true
		}

	}
}
