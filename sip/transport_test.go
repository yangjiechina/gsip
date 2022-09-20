package sip

import (
	"testing"
)

func TestUDPTransport(t *testing.T) {

	transport := UDPTransport{}
	err := transport.listen("127.0.0.1:5069")
	println(err)
}
