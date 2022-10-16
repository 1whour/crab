package utils

import (
	"fmt"
	"net"
)

// 获取没有绑定服务的地址(ip:port)
func GetUnusedAddr() string {
	startPort := 1000 //1000以下的端口很多时候需要root权限才能使用
	for port := startPort; port < 65535; port++ {
		addr := fmt.Sprintf("0.0.0.0:%d", port)
		l, err := net.Listen("tcp", addr)
		if err != nil {
			continue
		}
		l.Close()
		return addr
	}

	return ""
}
