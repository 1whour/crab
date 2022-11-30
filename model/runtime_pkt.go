package model

import (
	"strings"

	"github.com/antlabs/gstl/rwmap"
)

type RegisterRuntime struct {
	Whoami
	Ip string `json:"ip"` //绑定的gate
}

// runtime连接到gate，第一个包推带节点名
type Whoami struct {
	Name   string `json:"name"`
	Lambda bool   `json:"lambda"`
	Id     string `json:"id"`
}

// TODO: 通过http接口返回
type RuntimeResp struct {
	Kind    string `json:"kind"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

type RuntimeNode struct {
	RuntimeNode rwmap.RWMap[string, string]
	LambdaNode  rwmap.RWMap[string, string]
}

// 保存path信息
func (r *RuntimeNode) Store(key, value string) {
	if strings.Contains(key, LambdaKey) {
		r.LambdaNode.Store(key, value)
	} else {
		r.RuntimeNode.Store(key, value)
	}
}

func (r *RuntimeNode) Delete(key string) {
	if strings.Contains(key, LambdaKey) {
		r.LambdaNode.Delete(key)
	} else {
		r.RuntimeNode.Delete(key)
	}
}
