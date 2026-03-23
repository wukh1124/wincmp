package port

import (
	"fmt"
	"net"
	"strings"
)

type PortInfo struct {
	Service string
	Port    int
}

func CheckPorts(ports []PortInfo) []PortInfo {
	var inUse []PortInfo
	for _, p := range ports {
		if IsPortInUse(p.Port) {
			inUse = append(inUse, p)
		}
	}
	return inUse
}

func IsPortInUse(port int) bool {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "address already in use") {
			return true
		}
		return false
	}

	listener.Close()
	return false
}
