package client

import "net"

func getIpList() (rv []string, err error) {
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
