package sip

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	BranchPrefix = "z9hG4bK"
	SipVersion   = "SIP/2.0"
	SCHEME       = "scheme"
	URI          = "uri"
)

var (
	parsers map[string]HeaderParser
)

type HeaderParser func(name, value string) (Header, error)

func init() {
	parsers = map[string]HeaderParser{
		AcceptName:               parseIntOrStrHeader,
		AcceptEncodingName:       parseIntOrStrHeader,
		AcceptLanguageName:       parseIntOrStrHeader,
		AlertInfoName:            parseIntOrStrHeader,
		AllowName:                parseIntOrStrHeader,
		AuthenticationInfoName:   parseIntOrStrHeader,
		AuthorizationName:        parseAuthorizationHeader,
		CallIDName:               parseIntOrStrHeader,
		CallIDShortName:          parseIntOrStrHeader,
		CallInfoName:             parseIntOrStrHeader,
		ContactName:              parseAddressHeader,
		ContactShortName:         parseAddressHeader,
		ContentDispositionName:   parseIntOrStrHeader,
		ContentEncodingName:      parseIntOrStrHeader,
		ContentEncodingShortName: parseIntOrStrHeader,
		EventName:                parseEventHeader,
		ContentLanguageName:      parseIntOrStrHeader,
		ContentLengthName:        parseIntOrStrHeader,
		ContentLengthShortName:   parseIntOrStrHeader,
		ContentTypeName:          parseIntOrStrHeader,
		ContentTypeShortName:     parseCSeqHeader,
		CSeqName:                 parseCSeqHeader,
		DateName:                 parseIntOrStrHeader,
		ErrorInfoName:            parseIntOrStrHeader,
		ExpiresName:              parseIntOrStrHeader,
		FromName:                 parseAddressHeader,
		FromShortName:            parseAddressHeader,
		InReplyToName:            parseIntOrStrHeader,
		MaxForwardsName:          parseIntOrStrHeader,
		"MIME-Version":           parseIntOrStrHeader,
		MinExpiresName:           parseIntOrStrHeader,
		OrganizationName:         parseIntOrStrHeader,
		PriorityName:             parseIntOrStrHeader,
		ProxyAuthenticateName:    parseIntOrStrHeader,
		ProxyAuthorizationName:   parseIntOrStrHeader,
		ProxyRequireName:         parseIntOrStrHeader,
		RecordRouteName:          parseIntOrStrHeader,
		ReplyToName:              parseIntOrStrHeader,
		RequireName:              parseIntOrStrHeader,
		RetryAfterName:           parseIntOrStrHeader,
		RouteName:                parseAddressHeader,
		ServerName:               parseIntOrStrHeader,
		SubjectName:              parseIntOrStrHeader,
		SubjectShortname:         parseIntOrStrHeader,
		SubscriptionStateName:    parseSubscriptionStateHeader,
		SupportedName:            parseIntOrStrHeader,
		SupportedShortName:       parseIntOrStrHeader,
		TimestampName:            parseIntOrStrHeader,
		ToName:                   parseAddressHeader,
		ToShortName:              parseAddressHeader,
		UnsupportedName:          parseIntOrStrHeader,
		UserAgentName:            parseIntOrStrHeader,
		ViaName:                  parseViaHeader,
		ViaShortName:             parseViaHeader,
		WarningName:              parseIntOrStrHeader,
		WWWAuthenticateName:      parseWWWAuthenticateHeader,
	}
}

func parseUri(str string) (*SipUri, error) {
	if strings.HasPrefix(str, "sip:") {
		str = str[4:]
	} else if strings.HasPrefix(str, "sips:") {
		str = str[5:]
	} else {
		return nil, fmt.Errorf("the SIP URI prefix must be sips or sip")
	}

	index, offset := 0, len(str)

	//1.解析头 hname-hvalue
	//2.解析参数
	//3.解析userinfo和hostPort
	uri := SipUri{}
	index = strings.Index(str, "?")
	if index > 0 {
		if params, err := ParseParams(str[index+1:], "&"); err != nil {
			return nil, err
		} else {
			uri.Headers = params
			offset = index
		}
	}

	index = strings.Index(str, ";")
	if index > 0 {
		if params, err := ParseParams(str[index+1:offset], ";"); err != nil {
			return nil, err
		} else {
			uri.Params = params
			offset = index
		}
	}

	spearIndex := strings.Index(str[:offset], "@")
	if spearIndex > 0 {
		user, pwd := parseUserInfo(str[:spearIndex])
		uri.User = user
		uri.Password = pwd
	}

	hostPort := str[spearIndex+1 : offset]
	if host, port, err := ParseHostPort(hostPort); err != nil {
		return nil, err
	} else {
		uri.HostPort = HostPort{host, port}
		return &uri, nil
	}
}

func parseAddress(str string) (*Address, map[string]string, error) {
	l, r := -1, -1
	var uriStr string
	var paramsStr string
	var displayName string

	l = strings.Index(str, "<")
	if l >= 0 {
		r = strings.LastIndex(str, ">")
		if r < l {
			return nil, nil, fmt.Errorf("the URI format error:%s", str)
		}

		uriStr = str[l+1 : r]
		if l > 0 {
			displayName = str[:l]
		}

		if end := strings.Index(str[r+1:], ";"); end >= 0 {
			paramsStr = str[end+1:]
		}

	} else {
		index := strings.Index(str, "sip:")
		if index < 0 {
			return nil, nil, fmt.Errorf("the SIP URI prefix must be sips or sip:%s", str)
		}

		end := strings.Index(str[index:], ";")
		if end < 0 {
			uriStr = str[index:]
		} else {
			uriStr = str[index:end]
			paramsStr = str[end+1:]
		}
	}

	uri, err := parseUri(uriStr)
	if err != nil {
		return nil, nil, err
	}

	address := &Address{DisPlayName: displayName, Uri: uri}
	params := strings.Split(paramsStr, ";")

	m := make(map[string]string, 5)
	parse := func(k, v string) error {
		m[k] = v
		return nil
	}

	if err = ParseParams2(params[1:], parse); err != nil {
		return nil, nil, err
	}

	return address, m, nil
}

func parseRequestLine(str string) (*RequestLine, error) {
	split := strings.Split(str, " ")
	if len(split) != 3 {
		return nil, fmt.Errorf("the Request Line is invaild %s", str)
	}

	if uri, err := parseUri(split[1]); err != nil {
		return nil, err
	} else {
		return &RequestLine{split[0], uri, split[2]}, nil
	}
}

func parseStatusLine(str string) (*StatusLine, error) {
	split := strings.Split(str, " ")
	if len(split) < 3 {
		return nil, fmt.Errorf("the status Line is invaild %s", str)
	}

	if code, err := strconv.Atoi(split[1]); err != nil {
		return nil, err
	} else {
		return &StatusLine{split[0], code, strings.Join(split[2:], " ")}, nil
	}
}
func parseUserInfo(str string) (string, string) {
	if index := strings.Index(str, ":"); index > 0 {
		return str[:index], str[index+1:]
	} else {
		return str, ""
	}
}

func ParseHostPort(str string) (string, int, error) {

	index := strings.Index(str, ":")

	if index > 0 {
		port, err := strconv.Atoi(str[index+1:])
		return str[:index], port, err
	} else {
		return str, 0, nil
	}
}

func ParseParams(str string, separator string) (map[string]string, error) {

	nameV := strings.Split(str, separator)
	headers := make(map[string]string, len(nameV))

	for i := 0; i < len(nameV); i++ {
		k, v := SplitParamsByEqual(nameV[i])
		headers[k] = v
	}

	return headers, nil
}

func ParseParams2(parts []string, iterator func(k, v string) error) error {
	for _, part := range parts {
		k, v := SplitParamsByEqual(part)
		if err := iterator(k, v); err != nil {
			return err
		}
	}
	return nil
}

// SplitParams Use equal sign is default to parse param.
func SplitParams(str string, sign string) (string, string) {

	index := strings.Index(str, sign)
	if index < 0 {
		//return "", "", fmt.Errorf("the equal sign does not exist in the str:%s", str)
		return str, ""
	} else {
		return str[:index], str[index+1:]
	}
}

func SplitParamsByEqual(str string) (string, string) {
	return SplitParams(str, "=")
}

func parseViaHeader(_, str string) (Header, error) {
	parts := strings.Split(str, ";")
	if len(parts) < 2 {
		return nil, fmt.Errorf("the via format is invaild %s", str)
	}
	//先解析 版本/传输方式 sendby
	transportAndSendBy := strings.Split(parts[0], " ")
	if len(transportAndSendBy) != 2 || !strings.HasPrefix(transportAndSendBy[0], SipVersion) {
		return nil, fmt.Errorf("the via header parse failed %s", str)
	}

	protocol := strings.Split(transportAndSendBy[0], "/")
	if len(protocol) != 3 {
		return nil, fmt.Errorf("the via header protcol parse failed %s", str)
	}
	ip, port, err2 := ParseHostPort(transportAndSendBy[1])
	if err2 != nil {
		return nil, err2
	}

	via := &Via{sipVersion: SipVersion, transport: protocol[2], sendBy: HostPort{Host: ip, Port: port}}

	params := make(map[string]string, len(parts)-1)
	err := ParseParams2(parts[1:], func(k, v string) error {
		switch k {
		case "branch":
			via.branch = v
			break
		case "rport":
			if v != "" {
				p, err := strconv.Atoi(v)
				if err != nil {
					return err
				}
				via.rPort = p
			}
			break
		case "received":
			via.received = v
			break
		case "maddr":
			via.mAddr = v
			break
		case "ttl":
			if v != "" {
				p, err := strconv.Atoi(v)
				if err != nil {
					return err
				}
				via.ttl = p
			}
			break
		}

		params[k] = v
		return nil
	})

	if err != nil {
		return nil, err
	}

	//branch是必须存在的，匹配事务
	if via.branch == "" {
		return nil, fmt.Errorf("the VIA header must contain branch %s", str)
	}

	via.files = params
	return via, nil
}

func parseAddressValues(str string) ([]*Address, []map[string]string, error) {
	offset := 0
	isBrackets, isQuotes := false, false
	address := str + ","

	addresses := make([]*Address, 0, 1)
	params := make([]map[string]string, 0, 1)
	for i, e := range address {
		if e == '"' {
			isQuotes = !isQuotes
		} else if e == '<' && !isQuotes {
			isBrackets = true
		} else if e == '>' && !isQuotes {
			isBrackets = false
		} else if e == ',' && !isQuotes && !isBrackets {
			if adr, m, err := parseAddress(address[offset:i]); err != nil {
				return nil, nil, err
			} else {
				addresses = append(addresses, adr)
				params = append(params, m)
			}

			offset = i + 1
		}
	}

	return addresses, params, nil
}

// parseAddressHeader reference from https://github.com/ghettovoice/gosip
func parseAddressHeader(name, str string) (Header, error) {
	//from/to/contact/route/record-route/reply-to

	addresses, params, err := parseAddressValues(str)

	if err != nil {
		return nil, err
	}

	var header Header
	if "From" == name || "f" == name {
		header = &From{Address: addresses[0], Tag: params[0]["tag"]}
	} else if "To" == name || "t" == name {
		header = &To{Address: addresses[0], Tag: params[0]["tag"]}
	} else if "Contact" == name || "m" == name {

		contacts := make([]*Contact, 0, len(addresses))
		for i, addr := range addresses {
			contact := &Contact{Address: addr}
			for k, v := range params[i] {
				switch k {
				case "expires":
					if expires, err2 := strconv.Atoi(v); err2 != nil {
						return nil, err2
					} else {
						contact.Expires = expires
					}
					break
				case "q":
					if q, err2 := strconv.ParseFloat(v, 10); err2 != nil {
						return nil, err
					} else {
						contact.Q = float32(q)
					}
					break
				}
			}
			contacts = append(contacts, contact)
			header = &Contacts{Contacts: contacts}
		}

	} else if "Route" == name {
		var address []*SipUri
		for _, addr := range addresses {
			address = append(address, addr.Uri)
		}
		header = &Route{Address: address}
	}

	return header, nil
}

func parseCSeqHeader(_, str string) (Header, error) {
	split := strings.Split(str, " ")
	if len(split) != 2 {
		return nil, fmt.Errorf("the format of cSeq header is invaild %s", str)
	}

	if number, err := strconv.Atoi(split[0]); err != nil {
		return nil, err
	} else {
		return &CSeq{Number: number, Method: split[1]}, nil
	}
}

func parseAuth(str string, iterator func(k, v string) error) error {
	isQuotes, offset, parseOnce := false, 0, false

	for i, char := range str {
		if char == '"' {
			isQuotes = !isQuotes
			//last params
			parseOnce = !isQuotes && i == len(str)-1
		} else if (char == ',' || i == len(str)-1) && !isQuotes {
			parseOnce = true
		}

		if !parseOnce {
			continue
		}
		parseOnce = false

		var params string
		if char == ',' {
			params = str[offset:i]
		} else {
			params = str[offset:]
		}
		offset = i + 1

		split := strings.Split(params, "=")
		if len(split) != 2 {
			return fmt.Errorf("bad auth params :%s", params)
		}

		k := strings.TrimSpace(split[0])
		v := strings.TrimSpace(split[1])
		if strings.HasPrefix(v, "\"") || strings.HasSuffix(v, "\"") {
			v = strings.Trim(v, "\"")
		}

		if err := iterator(k, v); err != nil {
			return err
		}

	}

	return nil
}

func parseWWWAuthenticateHeader(_, str string) (Header, error) {
	index := strings.Index(str, " ")
	if index < 0 {
		return nil, fmt.Errorf("not Find digest in WWWAuthenticate %s", str)
	}

	schema := str[:index]
	if schema != "Digest" {

	}

	wwwAuthenticateHeader := NewWWWAuthenticateHeader()
	if err := parseAuth(str[index+1:], func(k, v string) error {
		//fmt.Printf("name: %s , value: %s", n, v)
		wwwAuthenticateHeader.SetParameter(k, v)
		return nil
	}); err != nil {
		return nil, err
	}

	return wwwAuthenticateHeader, nil
}

func parseAuthorizationHeader(_, str string) (Header, error) {
	index := strings.Index(str, " ")
	if index < 0 {
		return nil, fmt.Errorf("not Find digest in WWWAuthenticate %s", str)
	}

	schema := str[:index]
	if schema != "Digest" {

	}

	authorizationHeader := NewAuthorizationHeader()
	if err := parseAuth(str[index+1:], func(k, v string) error {
		//fmt.Printf("name: %s , value: %s", n, v)
		authorizationHeader.fields[k] = v
		if k == URI {
			if uri, err := parseUri(v); err == nil {
				authorizationHeader.SetUri(uri)
			} else {
				return err
			}
		} else {
			authorizationHeader.SetParameter(k, v)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return authorizationHeader, nil
}

func parseIntOrStrHeader(name, str string) (Header, error) {
	var header Header
	switch name {
	case CallIDName, CallIDShortName:
		callId := CallID(str)
		header = &callId
	case UserAgentName:
		agent := UserAgent(str)
		header = &agent
		break
	case MaxForwardsName:
		integer, err := strconv.Atoi(str)
		if err != nil {
			return nil, err
		}
		forwards := MaxForwards(integer)
		header = &forwards
	case ExpiresName:
		integer, err := strconv.Atoi(str)
		if err != nil {
			return nil, err
		}
		expires := Expires(integer)
		header = &expires
	case ContentLengthName, ContentLengthShortName:
		integer, err := strconv.Atoi(str)
		if err != nil {
			return nil, err
		}
		length := ContentLength(integer)
		header = &length
		break
	case ContentTypeName, ContentTypeShortName:
		t := ContentType(str)
		header = &t
		break
	case SubjectName, SubjectShortname:
		subject := Subject(str)
		header = &subject
		break
	default:
		header = &StrHeader{n: name, v: str}
	}

	return header, nil
}

func parseEventHeader(_, v string) (Header, error) {
	split := strings.Split(v, ";")
	e := &Event{Type: split[0]}
	if len(split) > 1 && strings.HasPrefix(split[1], "id") {
		_, v = SplitParamsByEqual(split[1])
		e.ID = v
	}

	return e, nil
}

func parseSubscriptionStateHeader(_, v string) (Header, error) {
	split := strings.Split(v, ";")
	header := &SubscriptionState{State: split[0]}

	if err := ParseParams2(split[1:], func(k, v string) error {
		switch k {
		case "reason":
			header.Reason = v
			break
		case "expires":
			header.Expires = v
			break
		case "retry-after":
			header.RetryAfter = v
			break
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return header, nil
}
