package sip

import (
	"bytes"
	"fmt"
)

type SipUri struct {
	User     string //userInfo part
	Password string //userInfo part. if the uri contains `@`, the user info cannot null.
	HostPort HostPort

	Params  map[string]string
	Headers map[string]string

	scheme string
}

func (uri *SipUri) Clone() *SipUri {
	clone := *uri
	if uri.Params != nil {
		clone.Params = deepCopy(uri.Params)
	}

	if uri.Headers != nil {
		clone.Headers = deepCopy(uri.Headers)
	}

	return &clone
}

func (uri *SipUri) ToString() string {
	var buffer bytes.Buffer
	if uri.HostPort.Host == "" {
		panic("the SIP URI must contain HOST")
	}

	buffer.WriteString("sip:")
	if uri.User != "" {
		buffer.WriteString(uri.User)
		if uri.Password != "" {
			buffer.WriteString(":")
			buffer.WriteString(uri.Password)
		}
		buffer.WriteString("@")
	}

	buffer.WriteString(uri.HostPort.ToString())

	if uri.Params != nil && len(uri.Params) > 0 {
		buffer.WriteString(";")
		params := mapToParamsStr(uri.Params, ";")
		buffer.WriteString(params)
	}

	if uri.Headers != nil && len(uri.Headers) > 0 {
		buffer.WriteString("?")
		params := mapToParamsStr(uri.Params, "&")
		buffer.WriteString(params)
	}

	return buffer.String()
}

func (uri *SipUri) GetScheme() string {
	return uri.scheme
}

func NewSipUri(user string, host string, port int) *SipUri {
	return &SipUri{User: user, HostPort: HostPort{Host: host, Port: port}}
}

type TelUri struct {
}

type HostPort struct {
	Host string
	Port int
}

func (h *HostPort) ToString() string {
	if h.Port > 0 {
		return fmt.Sprintf("%s:%d", h.Host, h.Port)
	} else {
		return h.Host
	}
}

type Address struct {
	DisPlayName string
	Uri         *SipUri
	Params      map[string]string
}

func (a *Address) Clone() *Address {
	clone := *a
	clone.Uri = a.Uri.Clone()
	return &clone
}

func NewAddress(uri *SipUri) *Address {
	return &Address{Uri: uri}
}

func CreateAddressWithDisplayName(name string, uri *SipUri) *Address {
	return &Address{DisPlayName: name, Uri: uri}
}

func deepCopy(src map[string]string) map[string]string {
	clone := make(map[string]string, len(src))
	for k, v := range src {
		clone[k] = v
	}

	return clone
}

func mapToParamsStr(m map[string]string, separator string) string {
	var buffer bytes.Buffer
	for k, v := range m {
		buffer.WriteString(k)
		if v != "" {
			buffer.WriteString("=")
			buffer.WriteString(v)
		}
		buffer.WriteString(separator)
	}

	return buffer.String()[0 : buffer.Len()-1]
}
