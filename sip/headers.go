package sip

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

const (
	AcceptName               = "Accept"
	AcceptEncodingName       = "Accept-Encoding"
	AcceptLanguageName       = "Accept-Language"
	AlertInfoName            = "Alert-Info"
	AllowName                = "Allow"
	AuthenticationInfoName   = "Authentication-Info"
	AuthorizationName        = "Authorization"
	AllowEventsName          = "Allow-Events"
	CallIDName               = "Call-ID"
	CallIDShortName          = "i"
	CallInfoName             = "Call-Info"
	ContactName              = "Contact"
	ContactShortName         = "m"
	ContentDispositionName   = "Content-Disposition"
	ContentEncodingName      = "Content-Encoding"
	ContentEncodingShortName = "e"
	ContentLanguageName      = "Content-Language"
	ContentLengthName        = "Content-Length"
	ContentLengthShortName   = "l"
	ContentTypeName          = "Content-Type"
	ContentTypeShortName     = "c"
	CSeqName                 = "CSeq"
	DateName                 = "Date"
	ErrorInfoName            = "Error-Info"
	EventName                = "Event"
	ExpiresName              = "Expires"
	FromName                 = "From"
	FromShortName            = "f"
	InReplyToName            = "In-Reply-To"
	MaxForwardsName          = "Max-Forwards"
	MimeVersionName          = "MIME-Version"
	MinExpiresName           = "Min-Expires"
	OrganizationName         = "Organization"
	PriorityName             = "Priority"
	ProxyAuthenticateName    = "Proxy-Authenticate"
	ProxyAuthorizationName   = "Proxy-Authorization"
	ProxyRequireName         = "Proxy-Require"
	RecordRouteName          = "Record-Route"
	ReplyToName              = "Reply-To"
	RequireName              = "Require"
	RetryAfterName           = "Retry-After"
	RouteName                = "Route"
	ServerName               = "Server"
	SubjectName              = "Subject"
	SubjectShortname         = "s"
	SupportedName            = "Supported"
	SupportedShortName       = "k"
	SubscriptionStateName    = "Subscription-State"
	TimestampName            = "Timestamp"
	ToName                   = "To"
	ToShortName              = "t"
	UnsupportedName          = "Unsupported"
	UserAgentName            = "User-Agent"
	ViaName                  = "Via"
	ViaShortName             = "v"
	WarningName              = "Warning"
	WWWAuthenticateName      = "WWW-Authenticate"
)

var (
	defaultContentLengthHeader = ContentLength(0)
	defaultMaxForwardsHeader   = MaxForwards(70)
)

type Header interface {
	Value() string
	Name() string
	Clone() Header
}

type StrHeader struct {
	n string
	v string
}

func (h *StrHeader) Value() string {
	return h.v
}

func (h *StrHeader) Name() string {
	return h.n
}

func (h *StrHeader) Clone() Header {
	return &*h
}

type Via struct {
	sipVersion string
	transport  string //UDP/TCP/TLS/SCTP
	branch     string
	sendBy     HostPort
	received   string //如果源IP和sendBy的IP不一样，响应的消息需要回复真实的IP到Received;
	rPort      int    //RFC3581 UAS发送响应消息根据包的源ip+via的SendBy的port来回复，但是很多UA都在nat后，所以增加了RPort
	ttl        int
	mAddr      string

	files map[string]string
}

func (via *Via) FindFiled(key string) (string, bool) {
	s, ok := via.files[key]
	return s, ok
}

func (via *Via) Value() string {
	return fmt.Sprintf("%s/%s %s;%s", SipVersion, via.transport, via.sendBy.ToString(), mapToParamsStr(via.files, ";"))
}

func (via *Via) Name() string {
	return ViaName
}

func (via *Via) SipVersion() string {
	return via.sipVersion
}
func (via *Via) Transport() string {
	return via.transport
}
func (via *Via) Branch() string {
	return via.branch
}
func (via *Via) SendBy() HostPort {
	return via.sendBy
}
func (via *Via) Received() string {
	return via.received
}
func (via *Via) RPort() int {
	return via.rPort
}
func (via *Via) TTL() int {
	return via.ttl
}
func (via *Via) MAddr() string {
	return via.mAddr
}

func (via *Via) setBranch(branch string) {
	via.branch = branch
	via.files["branch"] = branch
}
func (via *Via) setReceived(received string) {
	via.received = received
	via.files["received"] = received
}
func (via *Via) setRPort(port int) {
	via.rPort = port
	via.files["rport"] = strconv.Itoa(port)
}
func (via *Via) setTTL(ttl int) {
	via.ttl = ttl
	via.files["ttl"] = strconv.Itoa(ttl)
}
func (via *Via) setMAddr(addr string) {
	via.mAddr = addr
	via.files["maddr"] = addr
}
func (via *Via) Clone() Header {
	clone := *via
	if via.files != nil {
		clone.files = deepCopy(via.files)
	}

	return &clone
}

type From struct {
	Address *Address
	Tag     string
}

func NewFrom(user, host string, port int) *From {
	return &From{Address: NewAddress(NewSipUri(user, host, port))}
}

func (f *From) User() string {
	if f.Address != nil && f.Address.Uri != nil {
		return f.Address.Uri.User
	}
	return ""
}

func (f *From) Value() string {
	return fmt.Sprintf("%s<%s>;tag=%s", f.Address.DisPlayName, f.Address.Uri.ToString(), f.Tag)
}

func (f *From) Name() string {
	return FromName
}

func (f *From) Clone() Header {
	clone := *f
	clone.Address = f.Address.Clone()
	return &clone
}

type To From

func NewTo(user, host string, port int) *To {
	return &To{Address: NewAddress(NewSipUri(user, host, port))}
}

func (t *To) User() string {
	if t.Address != nil && t.Address.Uri != nil {
		return t.Address.Uri.User
	}
	return ""
}

func (t *To) Value() string {
	if t.Tag != "" {
		return fmt.Sprintf("%s<%s>;tag=%s", t.Address.DisPlayName, t.Address.Uri.ToString(), t.Tag)
	} else {
		return fmt.Sprintf("%s<%s>", t.Address.DisPlayName, t.Address.Uri.ToString())
	}

}

func (t *To) Name() string {
	return ToName
}

func (t *To) Clone() Header {
	clone := *t
	clone.Address = t.Address.Clone()
	return &clone
}

//type Contact AddressHeader

type Route struct {
	Address []*SipUri
}

func (r *Route) Name() string {
	return RouteName
}

func (r *Route) Value() string {
	var address []string
	for _, addr := range r.Address {
		address = append(address, "<"+addr.ToString()+">")
	}

	return strings.Join(address, ",")
}

func (r *Route) Clone() Header {
	address := make([]*SipUri, 0, len(r.Address))
	for _, uri := range r.Address {
		address = append(address, uri.Clone())
	}

	clone := *r
	clone.Address = address
	return &clone
}

type CallID string

func (c *CallID) Value() string {
	return string(*c)
}

func (c *CallID) Name() string {
	return CallIDName
}

func (c *CallID) Clone() Header {
	return &*c
}

func (c *CallID) ToString() string {
	return c.Value()
}

type UserAgent string

func (u UserAgent) Value() string {
	return string(u)
}

func (u *UserAgent) Name() string {
	return UserAgentName
}

func (u *UserAgent) Clone() Header {
	return &*u
}

func (u *UserAgent) ToString() string {
	return u.Value()
}

type Expires int

func (e *Expires) Value() string {
	return strconv.Itoa(int(*e))
}

func (e *Expires) Name() string {
	return ExpiresName
}

func (e *Expires) Clone() Header {
	return &*e
}
func (e *Expires) ToInt() int {
	return int(*e)
}

type MaxForwards int

func (m *MaxForwards) Value() string {
	return strconv.Itoa(int(*m))
}
func (m *MaxForwards) Name() string {
	return MaxForwardsName
}

func (m *MaxForwards) Clone() Header {
	return &*m
}

func (m *MaxForwards) ToInt() int {
	return int(*m)
}

type ContentLength int

func (c ContentLength) Value() string {
	return strconv.Itoa(int(c))
}

func (c *ContentLength) Name() string {
	return ContentLengthName
}

func (c *ContentLength) Clone() Header {
	return &*c
}

func (c *ContentLength) ToInt() int {
	return int(*c)
}

type ContentType string

func (c ContentType) Value() string {
	return string(c)
}

func (c *ContentType) Name() string {
	return ContentTypeName
}

func (c *ContentType) Clone() Header {
	return &*c
}

func (c *ContentType) ToString() string {
	return c.Value()
}

type Contact struct {
	Address *Address
	Q       float32
	Expires int
}

func (c *Contact) Value() string {
	var buffer bytes.Buffer
	if c.Address.DisPlayName != "" {
		buffer.WriteString(c.Address.DisPlayName)
	}
	buffer.WriteString("<")
	buffer.WriteString(c.Address.Uri.ToString())
	buffer.WriteString(">")
	if c.Q != 0 {
		buffer.WriteString(";q=")
		buffer.WriteString(fmt.Sprintf("%g", c.Q))
	}
	if c.Expires != 0 {
		buffer.WriteString(";expires=")
		buffer.WriteString(strconv.Itoa(c.Expires))
	}

	return buffer.String()
}

func (c *Contact) Name() string {
	return ContactName
}

func (c *Contact) Clone() Header {
	clone := *c
	clone.Address = c.Address.Clone()
	return &clone
}

type Contacts struct {
	Contacts []*Contact
}

func (c *Contacts) Value() string {
	var contacts []string
	for _, contact := range c.Contacts {
		contacts = append(contacts, contact.Value())
	}

	return strings.Join(contacts, ",")
}

func (c *Contacts) Name() string {
	return ContactName
}

func (c *Contacts) Clone() Header {
	contacts := make([]*Contact, 0, len(c.Contacts))
	for _, contact := range c.Contacts {
		contacts = append(contacts, contact)
	}
	clone := *c
	clone.Contacts = contacts

	return &clone
}

type CSeq struct {
	Number int
	Method string
}

func (c *CSeq) Value() string {
	return fmt.Sprintf("%d %s", c.Number, c.Method)
}

func (c *CSeq) Name() string {
	return CSeqName
}

func (c *CSeq) Clone() Header {
	return &*c
}

// WWWAuthenticate
//challenge         = "Digest" digest-challenge
//digest-challenge  = 1#( realm | [ domain ] | nonce | [ opaque ] |[ stale ] | [ algorithm ] | [ qop-options ] | [auth-param] )
type WWWAuthenticate struct {
	scheme string
	//realm     string
	//domain    string
	//nonce     string
	//opaque    string
	//stale     string //Nonce 过期标志 true/false
	//algorithm string //MD5/MD5-sess/token,default MD5
	//qop       string //[]string

	fields map[string]string
}

func NewWWWAuthenticateHeader() *WWWAuthenticate {
	header := &WWWAuthenticate{
		fields: make(map[string]string, 5),
	}
	header.SetScheme(DefaultSchema)
	header.SetAlgorithm(DefaultAlgorithm)

	return header
}

func (w *WWWAuthenticate) Scheme() string {
	return w.GetParameter("scheme")
}
func (w *WWWAuthenticate) Realm() string {
	return w.GetParameter("realm")
}
func (w *WWWAuthenticate) Domain() string {
	return w.GetParameter("domain")
}
func (w *WWWAuthenticate) Nonce() string {
	return w.GetParameter("nonce")
}
func (w *WWWAuthenticate) Opaque() string {
	return w.GetParameter("opaque")
}
func (w *WWWAuthenticate) Stale() string {
	return w.GetParameter("stale")
}
func (w *WWWAuthenticate) Algorithm() string {
	return w.GetParameter("algorithm")
}
func (w *WWWAuthenticate) Qop() string {
	return w.GetParameter("qop")
}

func (w *WWWAuthenticate) SetParameter(key, value string) {
	if key == SCHEME {
		w.scheme = value
	} else {
		w.fields[key] = value
	}
}

func (w *WWWAuthenticate) GetParameter(key string) string {
	if key == SCHEME {
		return w.scheme
	}
	return w.fields[key]
}

func (w *WWWAuthenticate) SetScheme(str string) {
	w.SetParameter("scheme", str)
}

func (w *WWWAuthenticate) SetRealm(str string) {
	w.SetParameter("realm", str)
}
func (w *WWWAuthenticate) SetDomain(str string) {
	w.SetParameter("domain", str)
}
func (w *WWWAuthenticate) SetNonce(str string) {
	w.SetParameter("nonce", str)
}
func (w *WWWAuthenticate) SetOpaque(str string) {
	w.SetParameter("opaque", str)
}
func (w *WWWAuthenticate) SetStale(str string) {
	w.SetParameter("stale", str)
}
func (w *WWWAuthenticate) SetAlgorithm(str string) {
	w.SetParameter("algorithm", str)
}
func (w *WWWAuthenticate) SetQop(str string) {
	w.SetParameter("qop", str)
}

func (w *WWWAuthenticate) Value() string {
	var buffer bytes.Buffer
	buffer.WriteString(w.Scheme())
	buffer.WriteString(" ")

	for k, v := range w.fields {
		if k == "algorithm" {
			buffer.WriteString(fmt.Sprintf("%s=%s,", k, v))
		} else {
			buffer.WriteString(fmt.Sprintf("%s=\"%s\",", k, v))
		}
	}

	return buffer.String()[:buffer.Len()-1]
}

func (w *WWWAuthenticate) Name() string {
	return WWWAuthenticateName
}

func (w *WWWAuthenticate) Clone() Header {
	clone := *w
	if w.fields != nil {
		clone.fields = deepCopy(w.fields)
	}

	return &clone
}

// Authorization
//credentials      = "Digest" digest-responseEvent
//digest-responseEvent  = 1#( username | realm | nonce | digest-uri | responseEvent | [ algorithm ] | [cnonce] | [opaque] | [message-qop] | [nonce-count]  | [auth-param] )
type Authorization struct {
	WWWAuthenticate
	//username string
	uri *SipUri
	//response string
	//cNonce   string //qop存在，Cnonce必须存在。qop不存在，Cnonce必须不存在
	//nc       string //int nonce-count qop存在，nc必须存在。qop不存在，nc必须不存在
}

func NewAuthorizationHeader() *Authorization {
	header := &Authorization{
		WWWAuthenticate: WWWAuthenticate{fields: make(map[string]string, 15)},
	}
	header.SetScheme(DefaultSchema)
	header.SetAlgorithm(DefaultAlgorithm)
	return header
}

func (a *Authorization) Username() string {
	return a.GetParameter("username")
}
func (a *Authorization) Uri() *SipUri {
	return a.uri
}
func (a *Authorization) Response() string {
	return a.GetParameter("response")
}
func (a *Authorization) CNonce() string {
	return a.GetParameter("cNonce")
}
func (a *Authorization) Nc() string {
	return a.GetParameter("nc")
}

func (a *Authorization) SetUsername(str string) {
	a.SetParameter("username", str)
}

func (a *Authorization) SetUri(uri *SipUri) {
	a.uri = uri /*uri.Clone()*/
	a.SetParameter("uri", uri.ToString())
}
func (a *Authorization) SetResponse(str string) {
	a.SetParameter("response", str)
}
func (a *Authorization) SetCNonce(str string) {
	a.SetParameter("cNonce", str)
}
func (a *Authorization) SetNc(str string) {
	a.SetParameter("nc", str)
}

func (a *Authorization) Value() string {
	return a.WWWAuthenticate.Value()
}

func (a *Authorization) Name() string {
	return AuthorizationName
}

func (a *Authorization) Clone() Header {
	clone := *a
	if a.WWWAuthenticate.fields != nil {
		clone.WWWAuthenticate.fields = deepCopy(a.WWWAuthenticate.fields)
	}
	if a.uri != nil {
		clone.uri = a.uri.Clone()
	}

	return &clone
}

type Event struct {
	Type string
	ID   string
}

func (e *Event) Value() string {
	if e.ID == "" {
		return e.Type
	} else {
		return fmt.Sprintf("%s;id=%s", e.Type, e.ID)
	}
}

func (e *Event) Name() string {
	return EventName
}

func (e *Event) Clone() Header {
	clone := *e
	return &clone
}

type SubscriptionState struct {
	State      string //active//pending/terminated
	Reason     string //deactivated//probation//rejected//timeout//giveup//noresource
	Expires    string
	RetryAfter string
}

func (s *SubscriptionState) Value() string {
	var buffer bytes.Buffer
	buffer.WriteString(s.State)
	if s.Reason != "" {
		buffer.WriteString(";reason=")
		buffer.WriteString(s.Reason)
	}
	if s.Expires != "" {
		buffer.WriteString(";expires=")
		buffer.WriteString(s.Expires)
	}
	if s.RetryAfter != "" {
		buffer.WriteString(";retry-after=")
		buffer.WriteString(s.RetryAfter)
	}
	return buffer.String()
}

func (s *SubscriptionState) Name() string {
	return SubscriptionStateName
}

func (s *SubscriptionState) Clone() Header {
	clone := *s
	return &clone
}

type Subject string

func (s *Subject) Value() string {
	return string(*s)
}

func (s *Subject) Name() string {
	return SubjectName
}

func (s *Subject) Clone() Header {
	subject := *s
	return &subject
}

type Allow string
