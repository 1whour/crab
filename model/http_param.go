package model

// 调度器发给执行器的包
type HTTP struct {
	Scheme  string      `yaml:"scheme"` //https还是http，默认是http
	Host    string      `yaml:"host"`   //是连接的主机名，默认是调度器的ip
	Port    int         `yaml:"port"`   //端口
	Path    string      `yaml:"path"`
	Method  string      `yaml:"method"`  //get post delete
	Querys  []NameValue `yaml:"querys"`  //存放查询字符串
	Headers []NameValue `yaml:"headers"` //存放http header
	Body    string      `yaml:"body"`    //存放body
}

type NameValue struct {
	Name  string
	Value string
}
