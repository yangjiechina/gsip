package main

import (
	"fmt"
	"gsip/examples"
	"gsip/sip"
	"time"
)

var (
	SipAgent       *SipServer
	xmlContentType = "Application/MANSCDP+xml"
)

type SipServer struct {
	listIP   string
	listPort int
	sipId    string
	password string

	stack *sip.Stack
	udp   *sip.ListeningPoint
	tcp   *sip.ListeningPoint
}

func (m *SipServer) start() {
	m.udp = &sip.ListeningPoint{IP: m.listIP, Port: m.listPort, Transport: sip.UDP}
	m.tcp = &sip.ListeningPoint{IP: m.listIP, Port: m.listPort, Transport: sip.TCP}
	m.stack = &sip.Stack{Listens: []*sip.ListeningPoint{m.udp, m.tcp}, EventListener: m,
		Option: sip.Option{
			UserAgent: "gsip test",
		}}
	err := m.stack.Start()
	if err != nil {
		panic(err)
	}

	//device := &Device{}
	//device.DeviceID = "34020000001110000001"
	//device.Transport = "UDP"
	//device.IP = m.listIP
	//device.Port = 5070
	//device.DoLive()
}

func (m *SipServer) getListeningPort(transport string) *sip.ListeningPoint {
	if sip.UDP == transport {
		return m.udp
	} else {
		return m.tcp
	}
}

func (m SipServer) newClientTransaction(transport string, request *sip.Request) (*sip.ClientTransaction, error) {
	return m.getListeningPort(transport).NewClientTransaction(request)
}

func (m *SipServer) createEmptyRequestMessage(method, requestUser, requestHost string, requestPort int, from, to string, transport string) *sip.Request {
	requestUri := sip.NewSipUri(requestUser, requestHost, requestPort)
	fromHeader := &sip.From{Address: sip.NewAddress(sip.NewSipUri(from, from[0:10], 0))}
	toHeader := &sip.To{Address: sip.NewAddress(sip.NewSipUri(to, to[0:10], 0))}
	listenPort := m.getListeningPort(transport)
	return listenPort.NewEmptyRequestMessage(method, requestUri, fromHeader, toHeader)
}

func (m *SipServer) createRequestMessage(requestUser, requestHost string, requestPort int, from, to string, transport string, contentType string, body []byte) *sip.Request {
	message := m.createEmptyRequestMessage(sip.MESSAGE, requestUser, requestHost, requestPort, from, to, transport)
	c := sip.ContentType(contentType)
	message.SetContent(&c, body)
	return message
}

func (m *SipServer) OnRegister(event *sip.RequestEvent) {
	request := event.Request

	var passwordCorrect bool

	if header := request.GetHeader(sip.AuthorizationName); header != nil {
		passwordCorrect = sip.DoAuthenticatePlainTextPassword(request, m.password)
		if passwordCorrect {
			fmt.Printf("密码正确\r\n")
		} else {
			fmt.Printf("密码错误\r\n")
		}
	}

	if !passwordCorrect {
		response := request.CreateResponse(sip.Unauthorized)
		sip.GenerateChallenge(response, m.sipId[0:10])
		event.ServerTransaction.SendResponse(response)
		return
	}

	via := request.Via()
	response := request.CreateResponse(sip.OK)
	uri := sip.NewSipUri(SipAgent.sipId, SipAgent.listIP, SipAgent.listPort)
	contact := sip.Contact{Address: sip.NewAddress(uri)}
	response.SetHeader(&contact)
	event.ServerTransaction.SendResponse(response)

	ip, port := request.GetRemoteHostPort()
	fromHeader := request.From()
	user := fromHeader.User()

	device := &Device{DeviceID: user, IP: ip, Port: port, Transport: via.Transport(), Channels: make(map[string]*Channel, 5)}
	deviceManager.Add(user, device)

	device.DoDeviceStatus()
	device.DoDeviceInfo()
	device.DoCatalog()
	device.DoLive()
	device.DoSubscribeMobilePosition()
}

func (m *SipServer) OnMessage(event *sip.RequestEvent) {
	var response *sip.Response
	response = event.Request.CreateResponse(sip.OK)
	event.ServerTransaction.SendResponse(response)
}

func (m *SipServer) OnRequest(event *sip.RequestEvent) {
	//fmt.Printf("OnRequest:%s", event.Request.ToString())

	request := event.Request
	switch event.Request.GetRequestMethod() {
	case sip.REGISTER:
		if header := request.Expires(); header != nil {
			if int(*header) == 0 {
				response := event.Request.CreateResponse(sip.OK)
				event.ServerTransaction.SendResponse(response)
				if device := deviceManager.Find(request.From().User()); device != nil {
					device.(*Device).OnLogout(event.Request)
				}
			} else {
				m.OnRegister(event)
			}
		}
		break
	case sip.MESSAGE:
		m.OnMessage(event)
		break
	case sip.NOTIFY:
		var response *sip.Response
		response = event.Request.CreateResponse(sip.OK)
		event.ServerTransaction.SendResponse(response)
		if find := deviceManager.Find(request.From().User()); find != nil {
			find.(*Device).OnNotify(event)
		}
		break
	}
}

func StartServer(config examples.ServerConfig) {
	SipAgent = &SipServer{
		listIP:   config.ListenIP,
		listPort: config.ListenPort,
		sipId:    config.SipId,
		password: config.Password,
	}
	SipAgent.start()

	for {
		var clientTransactions int
		var serverTransactions int
		var dialogs int
		i, i2, i3 := SipAgent.stack.Debug()
		clientTransactions += i
		serverTransactions += i2
		dialogs += i3
		fmt.Printf("%s clientTransactions:%d serverTransactions:%d dialogs:%d \r\n", time.Now().Format("2006-01-02T15:04:05"),
			clientTransactions,
			serverTransactions,
			dialogs)
		//runtime.GC()
		time.Sleep(5 * time.Second)
	}
}
