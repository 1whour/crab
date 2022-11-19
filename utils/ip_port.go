package utils

import (
	"errors"
	"fmt"
	"net"
)

// 获取没有绑定服务的地址(ip:port)
func GetUnusedPort(ip string) string {
	startPort := 1024 //1000以下的端口很多时候需要root权限才能使用
	for port := startPort; port < 65535; port++ {
		addr := fmt.Sprintf("%s:%d", ip, port)
		l, err := net.Listen("tcp", addr)
		if err != nil {
			continue
		}
		l.Close()
		return fmt.Sprintf("%d", port)
	}

	return ""
}

func GetIpList() (rv []string, err error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				rv = append(rv, ipnet.IP.String())
			}
		}
	}
	return
}

func GetIp() (rv string, err error) {
	ip, err := GetIpList()
	if err != nil {
		return "", err
	}
	if len(ip) == 0 {
		return "", errors.New("ip is empty")
	}
	return ip[0], nil
}

func GetUnusedAddr() string {
	ip, err := GetIp()
	if err != nil {
		return ""
	}

	return ip + ":" + GetUnusedPort(ip)
}
