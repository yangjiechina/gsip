package sip

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strings"
	"syscall"
)

var (
	UDP = "UDP"
	TCP = "TCP"
)

type transportHandler interface {
	onConnect(conn net.Conn)
	onDisconnect(conn net.Conn)
	onPacket(con net.Conn, isTCP bool, data []byte, length int)
}

func reusePortControl(network, address string, c syscall.RawConn) error {
	return c.Control(func(fd uintptr) {
		if runtime.GOOS != "darwin" {
			syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, 0x4, 1)
		}
	})
}

type ITransport interface {
	listen(address string) error
	recv(conn interface{})
	close()

	setHandler(transportHandler)
}

type transport struct {
	ctx    context.Context
	cancel context.CancelFunc

	handler transportHandler
}

func (t *transport) setHandler(handler transportHandler) {
	t.handler = handler
}

func (t *transport) close() {
	t.cancel()
}

type UDPTransport struct {
	transport
	udp []net.PacketConn
}

func (u *UDPTransport) listen(addr string) error {
	count := runtime.NumCPU()
	if runtime.GOOS == "darwin" {
		count = 1
	}

	u.ctx, u.cancel = context.WithCancel(context.Background())
	for i := 0; i < count; i++ {
		lc := net.ListenConfig{
			Control: reusePortControl,
		}
		socket, err := lc.ListenPacket(u.ctx, "udp", addr)
		if err != nil {
			return err
		}

		u.udp = append(u.udp, socket)
		go u.recv(socket)
	}

	return nil
}

func (u *UDPTransport) recv(conn interface{}) {
	udp := conn.(net.PacketConn)
	defer udp.Close()

	for u.ctx.Err() == nil {
		p := make([]byte, 1500)
		count, remote, err := udp.ReadFrom(p)
		if err != nil {
			fmt.Printf("udp recv failed: %v\n", err)
			continue
		}

		if u.handler != nil {
			c := &Conn{PacketConn: udp, local: udp.LocalAddr(), remote: remote}
			u.handler.onPacket(c, false, p, count)
		}
	}
}

type TCPServer struct {
	transport
}

func (t *TCPServer) listen(addr string) error {
	lc := net.ListenConfig{
		Control: reusePortControl,
	}

	t.ctx, t.cancel = context.WithCancel(context.Background())
	listener, err := lc.Listen(t.ctx, "tcp", addr)
	if err != nil {
		return err
	}

	go t.accept(listener.(*net.TCPListener))

	return nil
}

func (t *TCPServer) accept(listener *net.TCPListener) {
	for t.ctx.Err() == nil {
		tcp, err := listener.AcceptTCP()
		if err != nil {
			fmt.Printf("accept tcp connection failed: %v\n", err)
			continue
		}
		if t.handler != nil {
			t.handler.onConnect(tcp)
		}
		go t.recv(tcp)
	}
}

func (t *TCPServer) recv(conn interface{}) {
	tcp := conn.(*net.TCPConn)
	defer func() {
		if t.handler != nil {
			t.handler.onDisconnect(tcp)
		}
		tcp.Close()
	}()

	for t.ctx.Err() == nil {
		p := make([]byte, 4000)
		n, err := tcp.Read(p)
		if err != nil {
			fmt.Printf("tcp recv failed: %v\n", err)
			break
		}

		if t.handler != nil {
			t.handler.onPacket(tcp, true, p, n)
		}
	}
}

type TCPClient struct {
	transport
}

func (t *TCPClient) dial(addr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return conn, err
	}

	t.ctx, t.cancel = context.WithCancel(context.Background())
	if t.handler != nil {
		t.handler.onConnect(conn)
	}

	go t.recv(conn.(*net.TCPConn))
	return conn, nil
}

func (t *TCPClient) recv(tcp *net.TCPConn) {
	defer func() {
		if t.handler != nil {
			t.handler.onDisconnect(tcp)
		}
		tcp.Close()
	}()
	p := make([]byte, 16000)

	for t.ctx.Err() == nil {
		n, err := tcp.Read(p)
		if err != nil {
			fmt.Printf("tcp recv failed: %v\n", err)
			break
		}

		if t.handler.onPacket != nil {
			t.handler.onPacket(tcp, true, p, n)
		}
	}
}

func createServer(transport string, addr string) (ITransport, error) {
	var server ITransport
	switch strings.ToUpper(transport) {
	case "UDP":
		server = &UDPTransport{}
		break
	case "TCP":
		server = &TCPServer{}
		break
	}

	if server == nil {
		return nil, fmt.Errorf("unkown protocol %s ", transport)
	}

	err := server.listen(addr)

	return server, err
}
