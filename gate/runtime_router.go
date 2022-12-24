package gate

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/1whour/crab/model"
	"github.com/gin-gonic/gin"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	endKey         = model.RuntimeNodePrefix + "\xff\xff\xff\xff\xff\xff\xff\xff"
	startKeyPrefix = []byte("\x00")
	bytesEmpty     = []byte("")
)

type pageRuntime struct {
	Limit    int64  `form:"limit" json:"limit"`
	Page     int    `form:"page" json:"page"`
	Sort     string `form:"sort" json:"sort"`
	ID       string `form:"id" json:"id"`
	StartKey string `form:"start_key" json:"start_key"`
}

type runtimeNodeList struct {
	Total    int64  `json:"total"`
	Items    any    `json:"items"`
	StartKey string `json:"start_key"`
}

func (g *Gate) runtimeList(ctx *gin.Context) {

	p := pageRuntime{}
	if err := ctx.ShouldBindQuery(&p); err != nil {
		g.error2(ctx, 500, err.Error())
		return
	}

	// 默认10
	if p.Limit == 0 {
		p.Limit = 10
	}

	startKey := model.RuntimeNodePrefix
	if startKey == "" {
		startKey = p.StartKey
	}

	sortOrder := clientv3.SortAscend
	if strings.HasPrefix(p.Sort, "-") {
		sortOrder = clientv3.SortDescend
	}
	// 获取runtimeNode的值
	resp, err := defaultKVC.Get(g.ctx,
		startKey,
		clientv3.WithRange(endKey),
		clientv3.WithSort(clientv3.SortByKey, sortOrder),
		clientv3.WithLimit(p.Limit))
	if err != nil {
		g.error2(ctx, 500, err.Error())
		return
	}

	resp2, err2 := defaultKVC.Get(g.ctx, model.RuntimeNodePrefix, clientv3.WithCountOnly(), clientv3.WithPrefix())
	if err2 != nil {
		g.error2(ctx, 500, err2.Error())
		return
	}
	n := len(resp.Kvs)
	if len(p.ID) > 0 {
		n = 1
	}
	list := make([]model.RegisterRuntime, 0, n)

	for _, v := range resp.Kvs {
		var info model.RegisterRuntime
		err = json.Unmarshal(v.Value, &info)
		if err != nil {
			g.error2(ctx, 500, err.Error())
			return
		}

		g.Debug().Msgf("info.id %s, p.id %s", info.Id, p.ID)
		if len(p.ID) > 0 {
			if info.Id == p.ID {
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

	ctx.JSON(200, wrapData{Data: runtimeNodeList{
		Total:    resp2.Count,
		Items:    list,
		StartKey: startKey,
	}})
}
