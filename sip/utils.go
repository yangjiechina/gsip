package sip

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
)

func RandStr(length int) string {
	var buffer []byte
	if length == 6 {
		buffer = []byte{0, 0, 0, 0, 0, 0}
	} else if length == 12 {
		buffer = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	} else {
		buffer = make([]byte, length)
	}
	for i := 0; i < len(buffer); i++ {
		buffer[i] = byte(rand.Intn(255))
	}

	return hex.EncodeToString(buffer)
}

func generateCallId() string {
	return RandStr(24)
}

func generateBranchId() string {
	return BranchPrefix + "-" + RandStr(24)
}

func GenerateTag() string {
	return RandStr(12)
}

func generateTcpConnectKey(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}

func getHostPort(addr net.Addr) Hop {
	if _, ok := addr.(*net.UDPAddr); ok {
		return Hop{addr.(*net.UDPAddr).IP.String(), addr.(*net.UDPAddr).Port, UDP}
	} else {
		return Hop{addr.(*net.TCPAddr).IP.String(), addr.(*net.TCPAddr).Port, TCP}
	}
}
