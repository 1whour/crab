package gate

import (
	"bytes"
	"encoding/json"

	"github.com/1whour/crab/model"
	"github.com/gin-gonic/gin"
	"github.com/olekukonko/tablewriter"
)

var title = []string{"taskName", "status", "runtimeNode", "createTime", "updateTime", "runtimeID"}

type stateWithTaskRsp struct {
	pageStatus
	Task json.RawMessage `json:"task"`
}

// 响应的壳
type taskStatusList struct {
	Total int64 `json:"total"`
	Items any   `json:"items"`
}

// 构造status数据
// 内部使用接口， 直接返回格式化后的数据
// 标题如下
// taskName, status, runtimeNode, runtimeIP
func (g *Gate) status(ctx *gin.Context) {
	p := pageStatus{}

	err := ctx.ShouldBindQuery(&p)
	if err != nil {
		g.error2(ctx, 500, "bind query:"+err.Error())
		return
	}

	rv, count, err := g.statusTable.queryAndPage(p)
	if err != nil {
		g.error2(ctx, 500, "query data:"+err.Error())
		return
	}

	if p.Format == "table" {

		data := [][]string{}
		for _, v := range rv {
			one := []string{v.TaskName, v.Status, v.CreateTime.String(), v.UpdateTime.String(), v.RuntimeID}
			data = append(data, one)
		}

		var buf bytes.Buffer

		table := tablewriter.NewWriter(&buf)
		table.SetHeader(title)
		for _, d := range data {
			table.Append(d)
		}
		table.Render()

		ctx.String(200, buf.String())
	} else if p.Format == "json" {

		rsp := make([]stateWithTaskRsp, len(rv))
		for i, v := range rv {
			task, err := defautlClient.Get(g.ctx, model.FullGlobalTask(v.TaskName))
			if err != nil {
				g.Warn().Msgf("get state fail:%s", err)
				continue
			}
			rsp[i].pageStatus = v
			if len(task.Kvs) > 0 {
				rsp[i].Task = task.Kvs[0].Value
			}
		}
		ctx.JSON(200, wrapData{Data: taskStatusList{
			Total: count,
			Items: rsp,
		}})
	}
}
