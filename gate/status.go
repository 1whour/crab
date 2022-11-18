package gate

import (
	"bytes"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gnh123/scheduler/model"
	"github.com/olekukonko/tablewriter"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// 构造status数据
// 内部使用接口， 直接返回格式化后的数据
// 标题如下
// taskName, status, runtimeNode, runtimeIP
func (r *Gate) status(c *gin.Context) {
	format := c.Param("format")
	// 目前既既支持table格式数据
	if format == "table" {

		data := [][]string{}
		rspState, err := defaultKVC.Get(r.ctx, model.GlobalTaskPrefixState, clientv3.WithPrefix())
		if err != nil {
			c.String(500, err.Error())
			return
		}

		for _, kv := range rspState.Kvs {
			var d []string
			s, err := model.ValueToState(kv.Value)
			if err != nil {
				c.String(500, err.Error())
				return
			}
			taskName := model.TaskName(s.RuntimeNode)
			ip := ""
			if len(s.RuntimeNode) > 0 {
				rspState, err = defaultKVC.Get(r.ctx, s.RuntimeNode)
				if err != nil {
					c.String(500, err.Error())
					return
				}
				if len(rspState.Kvs) > 0 {
					ip = string(rspState.Kvs[0].Value)
				}
			}

			d = append(d, taskName, s.State, s.Action, s.RuntimeNode, fmt.Sprintf("%d", s.Successed), s.CreateTime.String(), s.UpdateTime.String(), ip)
			data = append(data, d)
		}

		var buf bytes.Buffer

		table := tablewriter.NewWriter(&buf)
		table.SetHeader([]string{"taskName", "status", "action", "runtimeNode", "successed", "createTime", "updateTime", "runtimeIP"})
		for _, d := range data {
			table.Append(d)
		}
		table.Render()

		c.String(200, buf.String())
	}
}
