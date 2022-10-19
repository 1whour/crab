package model

// runtime连接到gate，第一个包推带节点名
type Whoami struct {
	Name string `json:"name"`
}

type RuntimeResp struct {
	Kind    string `json:"kind"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Result  string `json:"result"`
}
