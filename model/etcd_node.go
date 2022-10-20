package model

import "fmt"

// gate node的信息
func FullGateNode(name string) string {
	return fmt.Sprintf("%s/%s", GateNodePrefix, name)
}

// 生成runtime node的信息
func FullRuntimeNode(runtimeName string) string {
	return fmt.Sprintf("%s/%s", RuntimeNodePrefix, runtimeName)
}
