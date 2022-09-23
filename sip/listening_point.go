package sip

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type ListeningPoint struct {
	IP        string
	Port      int
	Transport string

	transport   ITransport
	sipStack    *Stack
	tcpSessions *SafeMap
}

func (l *ListeningPoint) CreateViaHeader() *Via {
	m := make(map[string]string, 5)
	m["rport"] = ""
	m["received"] = ""
	return &Via{sipVersion: SipVersion, transport: strings.ToUpper(l.Transport), sendBy: HostPort{Host: l.IP, Port: l.Port}, files: m}
}

func (l *ListeningPoint) GetSendBy() string {
	return fmt.Sprintf("%s:%d", l.IP, l.Port)
}

func (l *ListeningPoint) onConnect(conn net.Conn) {
	tcpAddr := conn.RemoteAddr().(*net.TCPAddr)
	key := generateTcpConnectKey(tcpAddr.IP.String(), tcpAddr.Port)
	l.tcpSessions.Add(key, conn)
}

func (l *ListeningPoint) onDisconnect(conn net.Conn) {
	tcpAddr := conn.RemoteAddr().(*net.TCPAddr)
	key := generateTcpConnectKey(tcpAddr.IP.String(), tcpAddr.Port)
	l.tcpSessions.Remove(key)
}

func (l *ListeningPoint) onPacket(conn net.Conn, tcp bool, data []byte, length int) {
	err := processMessage(l, l.sipStack, conn, tcp, data, length)
	if err != nil {
		fmt.Printf("%s:%s", err.Error(), string(data[:length]))
	}
}

func (l *ListeningPoint) getConn(hop *Hop) (net.Conn, error) {
	if hop.Transport != strings.ToUpper(l.Transport) {
		panic("not find conn")
	}
	//如果是UDP，使用SIP监听的UDP端口
	//如果是TCP，查找链接是否已经存在，否则新建一个链接
	if strings.ToUpper(hop.Transport) == UDP {
		conn := l.transport.(*UDPTransport).udp[0]
		return &Conn{conn, conn.LocalAddr(), &net.UDPAddr{IP: net.ParseIP(hop.IP), Port: int(hop.Port)}}, nil
	} else {
		key := generateTcpConnectKey(hop.IP, hop.Port)
		if conn, b := l.tcpSessions.Find(key); b {
			return conn.(*net.TCPConn), nil
		} else {
			tcpClient := &TCPClient{}
			tcpClient.setHandler(l)
			return tcpClient.dial(fmt.Sprintf("%s:%d", hop.IP, hop.Port))
		}
	}
}

func (l *ListeningPoint) NewClientTransaction(request *Request) (*ClientTransaction, error) {
	return l.NewClientTransactionWithTimeout(request, l.sipStack.Option.RequestTimeout)
}

func (l *ListeningPoint) NewClientTransactionWithTimeout(request *Request, requestTimeout time.Duration) (*ClientTransaction, error) {

	if request.via == nil {
		viaHeader := l.CreateViaHeader()
		request.SetHeader(viaHeader)
	}

	if request.via.branch != "" {
		_, b := l.sipStack.findTransaction(request.GetTransactionId(), false)
		if b {
			return nil, fmt.Errorf("the client transction is exist")
		}
	} else {
		request.via.setBranch(generateBranchId())
	}

	_, b := l.sipStack.findTransaction(request.GetTransactionId(), false)
	if b {
		return nil, fmt.Errorf("the client transction is exist")
	}

	hop, err := findNextHop(request)
	if err != nil {
		return nil, err
	}

	tcp := strings.ToUpper(request.via.transport) == TCP
	invite := request.cSeq.Method == INVITE

	var stateMachine IStateMachine

	if invite {
		stateMachine = &InviteClientStateMachine{StateMachine: StateMachine{isTcp: tcp}}
	} else {
		stateMachine = &UnInviteClientStateMachine{StateMachine: StateMachine{isTcp: tcp}}
	}

	transactionId := request.GetTransactionId()
	t := &ClientTransaction{
		transaction: transaction{
			id:              transactionId,
			originalRequest: request,
			isInvite:        invite,
			stateMachine:    stateMachine,
			hop:             hop,
			sipStack:        l.sipStack,
			responseEvent:   make(chan *ResponseEvent, 1),
			ioError:         make(chan error, 1),
			txTimeout:       make(chan bool, 1),
			//txTerminated:    make(chan bool, 1),
		},
	}

	if requestTimeout > 0 {
		t.timeoutCtx, _ = context.WithTimeout(context.Background(), requestTimeout)
	}

	stateMachine.setTransaction(t)
	l.sipStack.addTransaction(transactionId, t, false)

	return t, nil
}
func (l *ListeningPoint) sendMessage(hop *Hop, msg Message) error {
	conn, err := l.getConn(hop)
	if err != nil {
		return err
	}

	_, err = conn.Write(msg.ToBytes())
	return err
}

func (l *ListeningPoint) SendRequest(msg *Request) error {
	if hop, err := findNextHop(msg); err != nil {
		return err
	} else {
		if hop.Transport != l.Transport {
			return fmt.Errorf("transport protocol does not match")
		}
		return l.sendMessage(hop, msg)
	}
}

func (l *ListeningPoint) SendResponse(msg *Response) error {
	via := msg.Via()
	if strings.ToUpper(via.transport) != strings.ToUpper(l.Transport) {
		return fmt.Errorf("transport protocol does not match")
	}

	host := via.sendBy.Host
	port := via.sendBy.Port
	if via.received != "" {
		host = via.received
	}
	if via.rPort != 0 {
		port = via.rPort
	}

	hop := &Hop{IP: host, Port: port, Transport: via.transport}
	return l.sendMessage(hop, msg)
}

func (l *ListeningPoint) newServerTransaction(request *Request, hop *Hop, conn net.Conn) (*ServerTransaction, error) {

	var stateMachine IStateMachine
	transactionId := request.GetTransactionId()
	tcp := request.GetTransport() == TCP
	invite := request.cSeq.Method == INVITE
	if invite {
		stateMachine = &InviteServerStateMachine{StateMachine: StateMachine{isTcp: tcp}}
	} else {
		stateMachine = &UnInviteServerStateMachine{StateMachine: StateMachine{isTcp: tcp}}
	}

	serverTransaction := &ServerTransaction{
		transaction{
			id:              transactionId,
			originalRequest: request,
			isInvite:        invite,
			stateMachine:    stateMachine,
			hop:             hop,
			conn:            conn,
			sipStack:        l.sipStack,
		},
	}

	stateMachine.setTransaction(serverTransaction)
	l.sipStack.addTransaction(transactionId, serverTransaction, true)

	return serverTransaction, nil
}

func (l *ListeningPoint) NewRequestMessage(method string, requestUri *SipUri, from *From, to *To, contentType *ContentType, body []byte) *Request {
	method = strings.ToUpper(method)
	request := &Request{
		message: message{
			line: &RequestLine{
				Method:     method,
				RequestUri: requestUri,
				SipVersion: SipVersion,
			},

			headers: make(map[string][]Header, 10),
		},
	}

	from.Tag = GenerateTag()
	callId := CallID(generateCallId())
	request.SetHeader(l.CreateViaHeader())
	request.SetHeader(from)
	request.SetHeader(to)
	request.SetHeader(&CSeq{Number: 1, Method: method})
	request.SetHeader(&callId)
	if contentType != nil {
		request.SetContent(contentType, body)
	} else {
		request.SetHeader(defaultContentLengthHeader.Clone())
	}

	if l.sipStack.Option.UserAgent != "" {
		request.SetUserAgent(l.sipStack.Option.UserAgent)
	}

	request.SetHeader(defaultMaxForwardsHeader.Clone())
	return request
}

func (l *ListeningPoint) NewEmptyRequestMessage(method string, requestUri *SipUri, from *From, to *To) *Request {
	return l.NewRequestMessage(method, requestUri, from, to, nil, nil)
}
