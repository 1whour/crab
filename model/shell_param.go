package model

// shell数据包
type Shell struct {
	Command string   `yaml:"command"` //命令
	Args    []string `yaml:"args"`    //命令分割成数组形式
}
