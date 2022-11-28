package model

import "fmt"

// gate node的信息
func FullGateNode(name string) string {
	return fmt.Sprintf("%s/%s", GateNodePrefix, name)
}

// 生成runtime node的信息
func FullRuntimeNode(who Whoami) string {
	runtimeName := who.Name
	if who.Lambda {
		return fmt.Sprintf("%s/%s", RuntimeNodeLambdaPrefix, runtimeName)

	}
	return fmt.Sprintf("%s/%s", RuntimeNodePrefix, runtimeName)
}
