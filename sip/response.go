package sip

import "fmt"

const (
	Trying                      = 100
	Ringing                     = 180
	CallIsBeingForwarded        = 181
	Queued                      = 182
	SessionProgress             = 183
	OK                          = 200
	Accepted                    = 202
	MultipleChoices             = 300
	MovedPermanently            = 301
	MovedTemporarily            = 302
	UseProxy                    = 305
	AlternativeService          = 380
	BadRequest                  = 400
	Unauthorized                = 401
	PaymentRequired             = 402
	Forbidden                   = 403
	NotFound                    = 404
	MethodNotAllowed            = 405
	NotAcceptable               = 406
	ProxyAuthenticationRequired = 407
	RequestTimeout              = 408
	Gone                        = 410
	RequestEntityTooLarge       = 413
	RequestURITooLong           = 414
	UnsupportedMediaType        = 415
	UnsupportedURIScheme        = 416
	BadExtension                = 420
	ExtensionRequired           = 421
	IntervalTooBrief            = 423
	TemporarilyUnavailable      = 480
	CallTransactionDoesNotExist = 481
	LoopDetected                = 482
	TooManyHops                 = 483
	AddressIncomplete           = 484
	Ambiguous                   = 485
	BusyHere                    = 486
	RequestTerminated           = 487
	NotAcceptableHere           = 488
	BadEvent                    = 489
	RequestPending              = 491
	Undecipherable              = 493
	ServerInternalError         = 500
	NotImplemented              = 501
	BadGateway                  = 502
	ServiceUnavailable          = 503
	ServerTim                   = 504
	VersionNotSupported         = 505
	MessageTooLarge             = 513
	BusyEverywhere              = 600
	Decline                     = 603
	DoesNotExistAnywhere        = 604
	SessionNotAcceptable        = 606
)

var reasons map[int]string

func init() {
	reasons = map[int]string{
		100: "Trying",
		180: "Ringing",
		181: "Call Is Being Forwarded",
		182: "Queued",
		183: "Session Progress",
		200: "OK",
		202: "Accepted",
		300: "Multiple Choices",
		301: "Moved Permanently",
		302: "Moved Temporarily",
		305: "Use Proxy",
		380: "Alternative Service",
		400: "Bad Request",
		401: "Unauthorized",
		402: "Payment Required",
		403: "Forbidden",
		404: "Not Found",
		405: "Method Not Allowed",
		406: "Not Acceptable",
		407: "Proxy Authentication Required",
		408: "Request Timeout",
		410: "Gone",
		413: "Request Entity Too Large",
		414: "Request-URI Too Long",
		415: "Unsupported Media Type",
		416: "Unsupported URI Scheme",
		420: "Bad Extension",
		421: "Extension Required",
		423: "Interval Too Brief",
		480: "Temporarily Unavailable",
		481: "Call transaction Does Not Exist",
		482: "Loop Detected",
		483: "Too Many Hops",
		484: "Address Incomplete",
		485: "Ambiguous",
		486: "Busy Here",
		487: "Request Terminated",
		488: "Not Acceptable Here",
		489: "Bad Event",
		491: "Request Pending",
		493: "Undecipherable",
		500: "Server Internal Error",
		501: "Not Implemented",
		502: "Bad Gateway",
		503: "Service Unavailable",
		504: "Server Tim",
		505: "Version Not Supported",
		513: "message Too Large",
		600: "Busy Everywhere",
		603: "Decline",
		604: "Does Not Exist Anywhere",
		606: "SESSION NOT ACCEPTABLE",
	}

}

type Response struct {
	message
}

func NewResponse() *Response {
	return &Response{message: message{headers: make(map[string][]Header, 10)}}
}

func (r *Response) GetStatusLine() *StatusLine {
	return r.line.(*StatusLine)
}

func (r *Response) SetStatusLine(statusLine *StatusLine) {
	r.line = statusLine
}

func (r *Response) GetStatusCode() int {
	return r.line.(*StatusLine).StatusCode
}

func (r *Response) GetReason() string {
	return r.line.(*StatusLine).Reason
}

func (r *Response) Clone() *Response {
	response := *r
	response.line = r.line.Clone()
	response.headers = make(map[string][]Header, len(r.headers))
	for _, headers := range r.headers {
		for _, header := range headers {
			response.AppendHeader(header)
		}
	}
	if r.body != nil {
		response.body = make([]byte, len(r.body))
		copy(response.body, r.body)
	}

	return &response
}

func (r *Response) CheckHeaders() error {
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

	if r.cSeq.Method == SUBSCRIBE && r.GetStatusCode() == OK && r.GetHeader(ExpiresName) == nil {
		return fmt.Errorf("the subscribe respsone must contain an expires header")
	}

	if isDialogCreated(r.cSeq.Method) && r.GetStatusCode() == OK && r.Contact() == nil {
		return fmt.Errorf("the %s response MUST contain a Contact header", r.cSeq.Method)
	}

	return nil
}
