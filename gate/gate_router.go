package gate

import (
	"bytes"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/1whour/crab/model"
	"github.com/gin-gonic/gin"
	"github.com/guonaihong/gout"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	endGateKey = model.GateNodePrefix + "\xff\xff\xff\xff\xff\xff\xff\xff"
)

// 请求
type pageGate struct {
	Limit    int64  `form:"limit" json:"limit"`
	Page     int    `form:"page" json:"page"`
	Sort     string `form:"sort" json:"sort"`
	ID       string `form:"id" json:"id"`
	StartKey string `form:"start_key" json:"start_key"`
}

// 响应的壳
type gateList struct {
	Total    int64  `json:"total"`
	Items    any    `json:"items"`
	StartKey string `json:"start_key"`
}

type gateItem struct {
	ID    string `json:"id"`
	IP    string `json:"ip"`
	Count int    `json:"count"`
}

func (g *Gate) gateCount(ctx *gin.Context) {
	g.Debug().Msgf("# runtime count %d", atomic.LoadInt32(&g.runtimeCount))
	ctx.String(200, fmt.Sprint(atomic.LoadInt32(&g.runtimeCount)))
}

func (g *Gate) gateList(ctx *gin.Context) {

	if defaultStore == nil {
		panic("defaultStore is nil")
	}
	// TODO 把分页逻辑抽离出来
	p := pageGate{}
	if err := ctx.ShouldBindQuery(&p); err != nil {
		g.error2(ctx, 500, err.Error())
		return
	}

	g.Debug().Msgf("gateList: pageGate:%v", p)
	// 默认10
	if p.Limit == 0 {
		p.Limit = 10
	}

	startKey := model.GateNodePrefix
	if p.ID == "" && p.StartKey != "" {
		startKey = p.StartKey
	}

	sortOrder := clientv3.SortAscend
	if strings.HasPrefix(p.Sort, "-") {
		sortOrder = clientv3.SortDescend
	}

	resp, err := defaultKVC.Get(g.ctx,
		startKey,
		clientv3.WithRange(endGateKey),
		clientv3.WithSort(clientv3.SortByKey, sortOrder),
		clientv3.WithLimit(p.Limit))
	if err != nil {
		g.error2(ctx, 500, err.Error())
		return
	}

	resp2, err2 := defaultKVC.Get(g.ctx, model.GateNodePrefix, clientv3.WithCountOnly(), clientv3.WithPrefix())
	if err2 != nil {
		g.error2(ctx, 500, err2.Error())
		return
	}

	n := len(resp.Kvs)
	if len(p.ID) > 0 {
		n = 1
	}

	list := make([]gateItem, 0, n)

	for _, v := range resp.Kvs {
		var info gateItem

		info.IP = string(v.Value)

		info.ID = model.TaskName(string(v.Key))
		count := int(0)
		err := gout.GET(info.IP + model.UI_GATE_COUNT).Debug(false).BindBody(&count).Do()
		if err != nil {
			g.Warn().Msgf("get fail:%s", err)
		}
		info.Count = count
		if len(p.ID) > 0 {
			g.Debug().Msgf("%s:%s", info.ID, p.ID)
			if info.ID == p.ID {
				list = append(list, info)
				break
			}
			continue
		}

		list = append(list, info)
	}

	if len(list) > 0 {
		startKey = string(bytes.Join([][]byte{resp.Kvs[0].Key, startKeyPrefix}, bytesEmpty))
	} else {
		startKey = ""
	}

	ctx.JSON(200, wrapData{Data: gateList{
		Total:    resp2.Count,
		Items:    list,
		StartKey: startKey,
	}})
}
