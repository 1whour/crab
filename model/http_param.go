package model

// 调度器发给执行器的包
type HTTP struct {
	Scheme  string      `yaml:"scheme" json:"scheme"` //https还是http，默认是http
	Host    string      `yaml:"host" json:"host"`     //是连接的主机名，默认是调度器的ip
	Port    int         `yaml:"port" json:"port"`     //端口
	Path    string      `yaml:"path" json:"path"`
	Method  string      `yaml:"method" json:"method"`   //get post delete
	Querys  []NameValue `yaml:"querys" json:"querys"`   //存放查询字符串
	Headers []NameValue `yaml:"headers" json:"headers"` //存放http header
	Body    string      `yaml:"body" json:"body"`       //存放body
}

type NameValue struct {
	Name  string
	Value string
}
