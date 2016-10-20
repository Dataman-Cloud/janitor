package util

import (
	"net"
)

func SliceContains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

func GetLocalIPs() []string {
	ipStrings := make([]string, 0)
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ipStrings
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipStrings = append(ipStrings, ipnet.IP.String())
			}
		}
	}
	return ipStrings
}
