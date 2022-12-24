package gate

import (
	"bytes"
	"encoding/json"

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

	// 获取runtimeNode的值
	resp, err := defaultKVC.Get(g.ctx,
		startKey,
		clientv3.WithRange(endKey),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
		clientv3.WithLimit(p.Limit))
	if err != nil {
		g.error2(ctx, 500, err.Error())
		return
	}

	resp2, err2 := defaultKVC.Get(g.ctx, model.RuntimeNodePrefix, clientv3.WithCountOnly())
	if err2 != nil {
		g.error2(ctx, 500, err2.Error())
		return
	}

	list := make([]model.RegisterRuntime, len(resp.Kvs))
	for i, v := range resp.Kvs {
		var info model.RegisterRuntime
		err = json.Unmarshal(v.Value, &info)
		if err != nil {
			g.error2(ctx, 500, err.Error())
			return
		}
		list[i] = info
	}

	if len(list) > 0 {
		startKey = string(bytes.Join([][]byte{resp.Kvs[0].Key, startKeyPrefix}, bytesEmpty))
	} else {
		startKey = ""
	}

	ctx.JSON(200, wrapData{Data: runtimeNodeList{
		Total:    int64(len(resp2.Kvs)),
		Items:    list,
		StartKey: startKey,
	}})
}
