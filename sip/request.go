package sip

import "fmt"

const (
	ACK       = "ACK"
	BYE       = "BYE"
	CANCEL    = "CANCEL"
	INVITE    = "INVITE"
	OPTIONS   = "OPTIONS"
	REGISTER  = "REGISTER"
	NOTIFY    = "NOTIFY"
	SUBSCRIBE = "SUBSCRIBE"
	MESSAGE   = "MESSAGE"
	REFER     = "REFER"
	INFO      = "INFO"
	PRACK     = "PRACK"
	UPDATE    = "UPDATE"
	PUBLISH   = "PUBLISH"
)

type Request struct {
	message
}

func NewRequest() *Request {
	return &Request{message{headers: make(map[string][]Header, 10)}}
}

func (r *Request) GetRequestLine() *RequestLine {
	return r.line.(*RequestLine)
}

func (r *Request) SetRequestLine(requestLine *RequestLine) {
	r.line = requestLine
}

func (r *Request) GetRequestMethod() string {
	return r.GetRequestLine().Method
}

func (r *Request) CheckHeaders() error {
	if r.via == nil {
		return fmt.Errorf("the VIA header is null for Request message")
	}
	if r.from == nil {
		return fmt.Errorf("the FROM header is null for Request message")
	}
	if r.to == nil {
		return fmt.Errorf("the TO header is null for Request message")
	}
	if r.callId == nil {
		return fmt.Errorf("the CALLID header is null for Request message")
	}
	if r.cSeq == nil {
		return fmt.Errorf("the CSEQ header is null for Request message")
	}
	//if r.max == nil {
	//	return fmt.Errorf("the MAXFORWARD header is null for Request message")
	//}

	line := r.GetRequestLine()

	if line.Method != r.cSeq.Method {
		return fmt.Errorf("CSEQ method mismatch with Request-Line")
	}

	if line.Method == SUBSCRIBE {
		if header := r.GetHeader(EventName); header == nil {
			return fmt.Errorf("the subscibe request must contain an event header")
		}
	} else if line.Method == NOTIFY {
		if header := r.GetHeader(SubscriptionStateName); header == nil {
			return fmt.Errorf("NOTIFY requests MUST contain a Subscription-State header")
		}
	}

	return nil
}

func (r *Request) RemoveTransactionTag() {
	if header := r.Via(); header != nil {
		header.setBranch("")
	}
}

func (r *Request) RemoveFromTag() {
	r.From().Tag = ""
}

func (r *Request) Clone() *Request {
	request := *r
	request.line = r.line.Clone()
	request.headers = make(map[string][]Header, len(r.headers))
	for _, headers := range r.headers {
		for _, header := range headers {
			request.AppendHeader(header)
		}
	}
	if r.body != nil {
		request.body = make([]byte, len(r.body))
		copy(request.body, r.body)
	}

	return &request
}

func (r *Request) CreateResponse(code int) *Response {
	reason, ok := reasons[code]
	if !ok {
		reason = "Unknown Status"
	}

	return r.CreateResponseWithReason(code, reason)
}

func (r *Request) CreateResponseWithReason(code int, reason string) *Response {
	response := &Response{message{line: &StatusLine{SipVersion, code, reason}, headers: make(map[string][]Header, 10)}}
	for _, header := range r.headers {
		switch header[0].Name() {
		case ViaName, ViaShortName, CallIDName, CallIDShortName, CSeqName, FromName, FromShortName, ToName, ToShortName, MaxForwardsName:
			response.SetHeader(header[0].Clone())
			break
		}
	}
	return response
}
