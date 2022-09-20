package sip

import "fmt"

type Line interface {
	IsRequest() bool
	ToString() string
	Clone() Line
}

type RequestLine struct {
	Method     string
	RequestUri *SipUri
	SipVersion string
}

func (r *RequestLine) IsRequest() bool {
	return true
}

func (r *RequestLine) ToString() string {
	return fmt.Sprintf("%s %s %s", r.Method, r.RequestUri.ToString(), r.SipVersion)
}

func (r *RequestLine) Clone() Line {
	requestLine := *r
	requestLine.RequestUri = r.RequestUri.Clone()
	return &requestLine
}

type StatusLine struct {
	SipVersion string
	StatusCode int
	Reason     string
}

func (s *StatusLine) CreateStatusLine(code int, reason string) *StatusLine {
	return &StatusLine{SipVersion: SipVersion, StatusCode: code, Reason: reason}
}

func (s *StatusLine) IsRequest() bool {
	return false
}

func (s *StatusLine) ToString() string {
	return fmt.Sprintf("%s %d %s", SipVersion, s.StatusCode, s.Reason)
}

func (s *StatusLine) Clone() Line {
	statusLine := *s
	return &statusLine
}
