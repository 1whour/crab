package gate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/1whour/crab/model"
	"github.com/antlabs/deepcopy"
	"github.com/gin-gonic/gin"
	"github.com/olekukonko/tablewriter"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var title = []string{"runtimeID", "taskName", "status", "action", "runtimeNode", "InRuntime", "createTime", "runtimeIP"}

const (
	endGlobalTaskKey = model.GlobalTaskPrefixState + "\xff\xff\xff\xff\xff\xff\xff\xff"
)

type stateRsp struct {
	RuntimeID  string          `json:"runtime_id"`
	TaskName   string          `json:"task_name"`
	Action     string          `json:"action"`
	State      string          `json:"state"`
	InRuntime  bool            `json:"in_runtime"`
	CreateTime time.Time       `json:"create_time"`
	Ip         string          `json:"ip"`
	Task       json.RawMessage `json:"task"`
}

// 响应的壳
type taskStatusList struct {
	Total    int64  `json:"total"`
	Items    any    `json:"items"`
	StartKey string `json:"start_key"`
}

// 构造status数据
// 内部使用接口， 直接返回格式化后的数据
// 标题如下
// taskName, status, runtimeNode, runtimeIP
func (g *Gate) status(ctx *gin.Context) {
	p := model.StatusRequest{}

	err := ctx.ShouldBindQuery(&p)
	if err != nil {
		g.error2(ctx, 500, "bind query:"+err.Error())
		return
	}

	// 默认10
	if p.Limit == 0 {
		p.Limit = 10
	}

	startKey := model.GlobalTaskPrefixState
	if p.ID == "" && p.StartKey != "" {
		startKey = p.StartKey
	}

	sortOrder := clientv3.SortAscend
	if strings.HasPrefix(p.Sort, "-") {
		sortOrder = clientv3.SortDescend
	}

	resp, err := defaultKVC.Get(g.ctx,
		startKey,
		clientv3.WithRange(endGlobalTaskKey),
		clientv3.WithSort(clientv3.SortByKey, sortOrder),
		clientv3.WithLimit(p.Limit))
	if err != nil {
		g.error2(ctx, 500, err.Error())
		return
	}

	if len(resp.Kvs) > 0 {
		g.Debug().Msgf("startKey, before(%s), after(%s)", startKey, string(resp.Kvs[len(resp.Kvs)-1].Key)+"\x00")
		startKey = string(resp.Kvs[len(resp.Kvs)-1].Key) + "\x00"
	}

	resp2, err2 := defaultKVC.Get(g.ctx, model.GlobalTaskPrefixState, clientv3.WithCountOnly(), clientv3.WithPrefix())
	if err2 != nil {
		g.error2(ctx, 500, err2.Error())
		return
	}

	n := len(resp.Kvs)
	if len(p.ID) > 0 {
		n = 1
	}

	list := make([]stateRsp, 0, n)

	data := [][]string{}
	for _, kv := range resp.Kvs {
		s, err := model.ValueToState(kv.Value)
		if err != nil {
			g.Debug().Msgf("status:%s", kv.Value)
			g.error2(ctx, 500, err.Error())
			return
		}
		ip := ""
		if len(s.RuntimeNode) > 0 {
			resp, err = defaultKVC.Get(g.ctx, s.RuntimeNode)
			if err != nil {
				g.error2(ctx, 500, err.Error())
				return
			}

			if len(resp.Kvs) > 0 {
				rnode := model.RegisterRuntime{}
				json.Unmarshal(resp.Kvs[0].Value, &rnode)
				ip = rnode.Ip
			}
		}

		if p.Format == "table" {
			one := []string{s.RuntimeID, s.TaskName, s.State, s.Action, s.RuntimeNode, fmt.Sprintf("%t", s.InRuntime), s.CreateTime.String(), s.UpdateTime.String(), ip}
			data = append(data, one)
		} else {
			rsp, err := defautlClient.Get(g.ctx, model.FullGlobalTask(s.TaskName))
			if err != nil {
				g.Warn().Msgf("get full data fail:%s, taskName:%s", err, s.TaskName)
				continue
			}

			var status stateRsp
			status.Ip = ip
			status.Task = rsp.Kvs[0].Value

			//g.Debug().Msgf("state rsp.createTime:%v, rsp.createtime:%v", s.CreateTime, status.CreateTime)
			deepcopy.Copy(&status, &s).Do()

			list = append(list, status)
		}
	}

	if p.Format == "table" {

		var buf bytes.Buffer

		table := tablewriter.NewWriter(&buf)
		table.SetHeader(title)
		for _, d := range data {
			table.Append(d)
		}
		table.Render()

		ctx.String(200, buf.String())
	} else if p.Format == "json" {

		ctx.JSON(200, wrapData{Data: taskStatusList{
			Total:    resp2.Count,
			Items:    list,
			StartKey: startKey,
		}})
	}
}
