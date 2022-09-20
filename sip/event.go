package sip

// RequestEvent Event和事务可以获得Dialog
//在首次创建Dialog,例如Invite响应2xx后，可以从事务中获取Dialog
//在后续的对话中请求，使用Event的dialog
type RequestEvent struct {
	Request           *Request
	Dialog            *Dialog
	ServerTransaction *ServerTransaction
}

type ResponseEvent struct {
	Response          *Response
	Dialog            *Dialog
	ClientTransaction *ClientTransaction
}
