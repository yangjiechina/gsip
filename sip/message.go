package sip

import (
	"bytes"
	"fmt"
	"net"
	"strings"
)

type Message interface {
	ToBytes() []byte
	ToString() string

	SetHeader(header Header)
	GetHeader(name string) []Header
	AppendHeader(header Header) error
	RemoveHeader(name string)
	SetContent(header *ContentType, body []byte)

	/**通常一个头域*/
	CallID() *CallID
	From() *From
	To() *To
	CSeq() *CSeq
	ContentLength() *ContentLength
	ContentType() *ContentType
	MaxForwards() *MaxForwards
	UserAgent() *UserAgent
	Expires() *Expires
	Via() *Via
	Event() *Event
	Contact() *Contact

	GetTransactionId() string
	CheckHeaders() error
	setBody(body []byte)
	setRemoteHostPort(ip string, port int)
	GetRemoteHostPort() (string, int)
	GetLocalHostPort() (ip string, port int)
	setLocalHostPort(ip string, port int)
}

type message struct {
	line    Line
	headers map[string][]Header
	body    []byte

	via           *Via
	from          *From
	to            *To
	callId        *CallID
	cSeq          *CSeq
	contentType   *ContentType
	contentLength *ContentLength
	userAgent     *UserAgent
	expires       *Expires
	maxForwards   *MaxForwards

	remoteIP   string
	remotePort int

	localIP   string
	localPort int
}

func (m *message) writeToBuffer2(buffer *bytes.Buffer, header Header) {
	buffer.Write([]byte(header.Name()))
	buffer.Write([]byte(": "))
	buffer.Write([]byte(header.Value()))
	buffer.Write([]byte("\r\n"))
}

func (m *message) writeToBuffer(buffer *bytes.Buffer, headers []Header) {
	if headers == nil {
		return
	}

	for _, header := range headers {
		m.writeToBuffer2(buffer, header)
	}
}

func (m *message) ToBytes() []byte {
	var buffer bytes.Buffer
	buffer.Write([]byte(m.line.ToString()))
	buffer.Write([]byte("\r\n"))

	if contentLengthHeader := m.GetHeader(ContentLengthName); contentLengthHeader == nil {
		m.SetHeader(&defaultContentLengthHeader)
	}
	//Via > Route > Record-Route > Proxy-Require > Max-Forwards > Proxy-Authorization > From > To > CallID > CSeq *** > ContentLength
	m.writeToBuffer(&buffer, m.GetHeader(ViaName))
	m.writeToBuffer(&buffer, m.GetHeader(RouteName))
	m.writeToBuffer(&buffer, m.GetHeader(RecordRouteName))
	m.writeToBuffer(&buffer, m.GetHeader(ProxyRequireName))
	if m.maxForwards != nil {
		m.writeToBuffer2(&buffer, m.maxForwards)
	}
	//m.writeToBuffer(buffer, m.GetHeader(ProxyAuthorization))
	m.writeToBuffer2(&buffer, m.from)
	m.writeToBuffer2(&buffer, m.to)
	m.writeToBuffer2(&buffer, m.callId)
	m.writeToBuffer2(&buffer, m.cSeq)

	// for headers
	for n, v := range m.headers {
		switch n {
		case ViaName, RouteName, RecordRouteName, ProxyRequireName, MaxForwardsName, FromName, ToName, CallIDName, CSeqName, ContentLengthName:
			break
		default:
			m.writeToBuffer(&buffer, v)
		}
	}

	m.writeToBuffer2(&buffer, m.ContentLength())
	buffer.Write([]byte("\r\n"))
	if m.body != nil {
		buffer.Write(m.body)
	}

	return buffer.Bytes()
}

func (m *message) ToString() string {
	return string(m.ToBytes())
}

func (m *message) SetHeader(header Header) {
	headers := []Header{header}
	m.headers[header.Name()] = headers
	switch header.Name() {
	case ViaName, ViaShortName:
		m.via = header.(*Via)
		break
	case FromName /*, FROM_SHORT_NAME*/ :
		m.from = header.(*From)
		break
	case ToName /*, TO_SHORT_NAME*/ :
		m.to = header.(*To)
		break
	case CallIDName /*, CALL_ID_SHORT_NAME*/ :
		m.callId = header.(*CallID)
		break
	case CSeqName:
		m.cSeq = header.(*CSeq)
		break
	case ContentTypeName:
		m.contentType = header.(*ContentType)
		break
	case ContentLengthName:
		m.contentLength = header.(*ContentLength)
		break
	case UserAgentName:
		m.userAgent = header.(*UserAgent)
		break
	case ExpiresName:
		m.expires = header.(*Expires)
		break
	case MaxForwardsName:
		m.maxForwards = header.(*MaxForwards)
		break
	}
}

func (m *message) AppendHeader(header Header) error {
	if headers, ok := m.headers[header.Name()]; ok {
		switch header.Name() {
		case FromName, FromShortName, ToName, ToShortName, CallIDName, CallIDShortName, CSeqName, MaxForwardsName, ExpiresName, UserAgentName, ContentTypeName, ContentTypeShortName, ContentLengthName, ContentLengthShortName:
			if headers[0].Name() == header.Name() {
				return fmt.Errorf("multiple header field rows are not appropriate in the %s header", header.Name())
			}
		}
		headers = append(headers, header)
		m.headers[header.Name()] = headers
	} else {
		m.SetHeader(header)
	}

	return nil
}

func (m *message) GetHeader(name string) []Header {
	return m.headers[name]
}

func (m *message) RemoveHeader(name string) {
	delete(m.headers, name)
}

func (m *message) SetContent(header *ContentType, body []byte) {
	m.SetHeader(header)
	c := ContentLength(len(body))
	m.SetHeader(&c)
	m.body = body
}

func (m *message) CallID() *CallID {
	return m.callId
}

func (m *message) From() *From {
	return m.from
}

func (m *message) To() *To {
	return m.to
}

func (m *message) CSeq() *CSeq {
	return m.cSeq
}

func (m *message) ContentLength() *ContentLength {
	return m.contentLength
}

func (m *message) ContentType() *ContentType {
	return m.contentType
}

func (m *message) MaxForwards() *MaxForwards {
	return m.maxForwards
}

func (m *message) UserAgent() *UserAgent {
	return m.userAgent
}

func (m *message) Expires() *Expires {
	return m.expires
}

func (m *message) Via() *Via {
	return m.via
}

func (m *message) Event() *Event {
	if header := m.GetHeader(EventName); header != nil {
		return header[0].(*Event)
	}

	return nil
}

func (m *message) Contact() *Contact {
	if header := m.GetHeader(ContactName); header != nil {
		if contacts, ok := header[0].(*Contacts); ok {
			return contacts.Contacts[0]
		} else {
			return header[0].(*Contact)
		}
	}

	return nil
}

func (m *message) GetTransactionId() string {
	via := m.Via()
	cseq := m.CSeq()
	if cseq.Method == CANCEL {
		return via.branch + ":" + cseq.Method
	} else {
		return via.branch
	}
}

// GetDialogId
// call identifier : remote tag : remote tag
func (m *message) GetDialogId(uas bool) string {
	fromHeader := m.From()
	toHeader := m.To()
	callIdHeader := m.CallID()
	if uas {
		return fmt.Sprintf("%s:%s:%s", string(*callIdHeader), fromHeader.Tag, toHeader.Tag)
	} else {
		return fmt.Sprintf("%s:%s:%s", string(*callIdHeader), toHeader.Tag, fromHeader.Tag)
	}
}

func (m *message) setBody(body []byte) {
	m.body = body
}

func (m *message) GetRemoteHostPort() (string, int) {
	return m.remoteIP, m.remotePort
}

func (m *message) setRemoteHostPort(ip string, port int) {
	m.remoteIP = ip
	m.remotePort = port
}

func (m *message) GetTransport() string {
	return m.Via().transport
}

func (m *message) Content() []byte {
	return m.body
}

func (m *message) SetFromTag(tag string) {
	m.From().Tag = tag
}

func (m *message) SetToTag(tag string) {
	m.To().Tag = tag
}

func (m *message) SetExpires(expires int) {
	if header := m.Expires(); header != nil {
		*header = Expires(expires)
	} else {
		e := Expires(expires)
		m.SetHeader(&e)
	}
}

func (m *message) SetMaxForward(max int) {
	if header := m.MaxForwards(); header != nil {
		*header = MaxForwards(max)
	} else {
		e := MaxForwards(max)
		m.SetHeader(&e)
	}
}

func (m *message) SetUserAgent(agent string) {
	if header := m.UserAgent(); header != nil {
		*header = UserAgent(agent)
	} else {
		e := UserAgent(agent)
		m.SetHeader(&e)
	}
}

func (m *message) GetLocalHostPort() (ip string, port int) {
	return m.localIP, m.localPort
}

func (m *message) setLocalHostPort(ip string, port int) {
	m.localIP = ip
	m.localPort = port
}

func processMessage(listeningPoint *ListeningPoint, stack *Stack, conn net.Conn, tcp bool, data []byte, length int) error {
	msg, isRequest, err := parseMessage(data, length)
	if err != nil {
		return err
	}
	viaHeader := msg.Via()
	if (viaHeader.transport == UDP && tcp) || (viaHeader.transport == TCP && !tcp) {
		return fmt.Errorf("the transport protocol of VIA header is not the same as that in the network layer")
	}

	hop := getHostPort(conn.RemoteAddr())
	msg.setRemoteHostPort(hop.IP, hop.Port)
	msg.setLocalHostPort(listeningPoint.IP, listeningPoint.Port)

	if stack.EventInterceptor != nil {
		if isRequest {
			stack.EventInterceptor.OnRequest(msg.(*Request))
		} else {
			stack.EventInterceptor.OnResponse(msg.(*Response))
		}
		return nil
	}

	if isRequest && stack.EventListener == nil {
		return fmt.Errorf("event listener are nil")
	}

	transactionId := msg.GetTransactionId()
	if isRequest {
		request := msg.(*Request)
		if _, isRequest = viaHeader.FindFiled("rport"); isRequest {
			viaHeader.setRPort(hop.Port)
			viaHeader.setReceived(hop.IP)
		}
		//create server transaction
		t, _ := stack.findTransaction(transactionId, true)
		if t == nil {
			//if ACK == request.GetRequestMethod() {
			//	return fmt.Errorf("the server t does not exist")
			//}
			t, err = listeningPoint.newServerTransaction(request, &hop, conn)
		}

		t.(*ServerTransaction).processRequest(request)
	} else {
		response := msg.(*Response)
		if client, b := stack.findTransaction(transactionId, false); b {
			client.(*ClientTransaction).processResponse(response)
		} else {
			return fmt.Errorf("the client transaction does not exist")
		}
	}

	return nil
}

func parseMessage(data []byte, length int) (Message, bool, error) {
	first, isRequest := true, false
	offset, index := 0, 0
	var msg Message
	for index < length {
		for index < length && data[index] != '\r' {
			index++
		}

		l := string(data[offset:index])

		if first {
			//responseEvent
			if strings.HasPrefix(l, SipVersion) {
				if statusLine, err := parseStatusLine(l); err != nil {
					return nil, false, err
				} else {
					msg = &Response{
						message: message{
							line:    statusLine,
							headers: make(map[string][]Header, 10),
						},
					}
				}

			} else {
				isRequest = true
				//Request
				if requestLine, err := parseRequestLine(l); err != nil {
					return nil, false, err
				} else {
					msg = &Request{
						message: message{
							line:    requestLine,
							headers: make(map[string][]Header, 10),
						},
					}

				}
			}

			first = false
		} else {

			i := strings.Index(l, ":")
			if i < 0 || len(l) == i+1 {
				break
			}

			hName := l[:i]
			var hValue string
			if l[i+1:i+2] == " " {
				hValue = l[i+2:]
			} else {
				hValue = l[i+1:]
			}

			if strings.HasPrefix(hName, " ") || strings.HasSuffix(hName, " ") {
				hName = strings.TrimSpace(hName)
			}
			if strings.HasPrefix(hValue, " ") || strings.HasSuffix(hValue, " ") {
				hValue = strings.TrimSpace(hValue)
			}

			if hValue == "" {
				return nil, false, fmt.Errorf("bad message. the %s header value is empty", hValue)
			}

			if parser, ok := parsers[hName]; !ok {
				return nil, false, fmt.Errorf("unknow header:%s", hName)
			} else {
				if header, err := parser(hName, hValue); err != nil {
					return nil, false, err
				} else {
					if err := msg.AppendHeader(header); err != nil {
						return nil, false, err
					}
				}
			}
		}

		index += 2
		offset = index

		//消息头结束
		if index+1 < length && data[index] == '\r' && data[index+1] == '\n' {
			break
		}
	}

	if msg == nil {
		return nil, false, fmt.Errorf("the message parse failed")
	}

	if err := msg.CheckHeaders(); err != nil {
		return nil, false, err
	}

	if header := msg.ContentLength(); header != nil && *header != 0 {
		contentLength := int(*header)
		index += 2
		if index+contentLength > length {
			return nil, false, fmt.Errorf("the packet size is smaller than content length")
		}

		msg.setBody(data[index : index+contentLength])
	}

	return msg, isRequest, nil
}
