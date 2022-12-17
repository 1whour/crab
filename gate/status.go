package gate

import (
	"bytes"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/1whour/ktuo/model"
	"github.com/olekukonko/tablewriter"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/exp/slices"
)

var title = []string{"taskName", "status", "action", "runtimeNode", "InRuntime", "createTime", "updateTime", "runtimeIP"}

// 构造status数据
// 内部使用接口， 直接返回格式化后的数据
// 标题如下
// taskName, status, runtimeNode, runtimeIP
func (r *Gate) status(c *gin.Context) {
	req := model.StatusRequest{}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	// 目前既既支持table格式数据
	if req.Format == "table" {

		data := [][]string{}
		rspState, err := defaultKVC.Get(r.ctx, model.GlobalTaskPrefixState, clientv3.WithPrefix())
		if err != nil {
			c.String(500, err.Error())
			return
		}

		for _, kv := range rspState.Kvs {
			s, err := model.ValueToState(kv.Value)
			if err != nil {
				c.String(500, err.Error())
				return
			}
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
			one := []string{s.TaskName, s.State, s.Action, s.RuntimeNode, fmt.Sprintf("%t", s.InRuntime), s.CreateTime.String(), s.UpdateTime.String(), ip}
			if len(req.Filter) > 0 {
				for i := 0; i < len(req.Filter) && len(req.Filter)%2 == 0; i += 2 {
					pos := slices.Index(title, req.Filter[i])
					if pos != -1 {
						if req.Filter[i+1] != one[pos] {
							goto next
						}
					}
				}
			}
			data = append(data, one)
		next:
		}

		var buf bytes.Buffer

		table := tablewriter.NewWriter(&buf)
		table.SetHeader(title)
		for _, d := range data {
			table.Append(d)
		}
		table.Render()

		c.String(200, buf.String())
	}
}
