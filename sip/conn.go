package sip

import (
	"net"
)

type Conn struct {
	net.PacketConn
	local  net.Addr
	remote net.Addr
}

func (c *Conn) Read(b []byte) (int, error) {
	if n, _, err := c.ReadFrom(b); err != nil {
		return n, err
	} else {
		return n, err
	}
}

func (c *Conn) Write(b []byte) (int, error) {
	return c.WriteTo(b, c.remote)
}

func (c *Conn) LocalAddr() net.Addr {
	if c.local == nil {
		return c.PacketConn.LocalAddr()
	}
	return c.local
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.remote
}
