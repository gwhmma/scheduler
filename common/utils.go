package common

import (
	"net"
	"strings"
)

// 返回本机的IP
func GetLocalIP() (string, error) {
	// 获取所有地址
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	// 取第一个非localhost的网卡ip
	for _, addr := range addrs {
		// ipv4 ipv6  不是环回地址
		if ipNet, isIpNet := addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			// 只需要ipv4
			if ipNet.IP.To4() != nil {
				return ipNet.IP.To4().String(), nil
			}
		}
	}

	return "", ERR_NO_LOCAL_IP_FOUND
}

func ExtractName(s, prefix string) string {
	return strings.TrimPrefix(s, prefix)
}