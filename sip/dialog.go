package sip

import (
	"fmt"
	"strings"
)

const (
	dialogStateEarly      = 1 //收到临时应答
	dialogStateConfirmed  = 2 //2xx应答 其余终结应答dialog终结
	dialogStateTerminated = 3
)

// DialogId callId:remoteTag:localTag
type DialogId string

func (d DialogId) CallId() string {
	return strings.Split(string(d), ":")[0]
}
func (d DialogId) RemoteTag() string {
	return strings.Split(string(d), ":")[1]
}
func (d DialogId) LocalTag() string {
	return strings.Split(string(d), ":")[2]
}

type seqNumber int

func newSeqNumber(i int) *seqNumber {
	number := seqNumber(i)
	return &number
}

func (c *seqNumber) greater(i int) bool {
	return int(*c) > i
}

func (c *seqNumber) less(i int) bool {
	return int(*c) < i
}

func (c *seqNumber) increase() {
	*c += 1
}

func (c *seqNumber) setValue(i int) {
	*c = seqNumber(i)
}

type Dialog struct {
	dialogId        DialogId   //UAS:callId+to tag+ from tag
	localSeqNumber  *seqNumber //UAS:null
	remoteSeqNumber *seqNumber //UAS:请求的Cseq number
	localUri        *SipUri    //UAS:TO uri
	remoteUri       *SipUri    //UAS:From uri
	remoteTarget    *SipUri    //UAS:设置为请求Contact头的uri
	secure          bool       //UAS:请求通过TLS传输 并且Request-uri是sips uri. secure设置为true
	isUAC           bool
	state           int
	sipStack        *Stack
	listeningPoint  *ListeningPoint
	routeSet        []*SipUri
	via             *Via
	//route set
}

//type Dialog struct {
//	dialogId         string //UAC:callId+from tag+ to tag
//	localCseqNumber  int    //UAC:请求的Cseq number
//	remoteCseqNumber int    //UAC:null
//	localUri         string //UAC:From uri
//	remoteUri        string //UAC:To uri
//	remoteTarget     string //UAC:设置为应答Contact头的uri
//	secure           bool   //UAC:请求通过TLS传输 并且Request-uri是sips uri. secure设置为true
//}
//re-Invite 刷新target要修改 remoteTag

func createDialog(stack *Stack, listeningPoint *ListeningPoint, request *Request, response *Response, uas bool) *Dialog {
	var dialog *Dialog
	cSeqHeader := request.CSeq()
	fromHeader := request.From()
	toHeader := response.To()
	number := seqNumber(cSeqHeader.Number)

	if uas {
		contactHeader := request.Contact()
		dialog = &Dialog{
			remoteTarget:    contactHeader.Address.Uri,
			remoteSeqNumber: &number,
			dialogId:        DialogId(response.GetDialogId(true)),
			remoteUri:       fromHeader.Address.Uri,
			localUri:        toHeader.Address.Uri,
			secure:          false,
		}

	} else {
		contactHeader := response.Contact()
		dialog = &Dialog{
			remoteTarget:   contactHeader.Address.Uri,
			localSeqNumber: &number,
			dialogId:       DialogId(response.GetDialogId(false)),
			remoteUri:      toHeader.Address.Uri,
			localUri:       fromHeader.Address.Uri,
			secure:         false,
		}
	}

	dialog.via = response.via
	dialog.sipStack = stack
	dialog.listeningPoint = listeningPoint
	return dialog
}

func (d *Dialog) SendAck(request *Request) error {
	return d.listeningPoint.SendRequest(request)
}

func (d *Dialog) CreateAck(cSeqNumber int) *Request {
	ack := d.createRequest(ACK, cSeqNumber)
	ack.Via().setBranch(generateBranchId())
	return ack
}

func (d *Dialog) createRequest(method string, cSeqNumber int) *Request {
	//CSeq的seqNumber在Cancel和ACK请求需要和原始请求保持一致，其余请求依次递增
	//remote target 作为request uri
	toHeader := &To{Address: &Address{Uri: d.remoteUri}, Tag: d.dialogId.RemoteTag()}
	fromHeader := &From{Address: &Address{Uri: d.localUri}, Tag: d.dialogId.LocalTag()}
	callIdHeader := CallID(d.dialogId.CallId())

	cSeqHeader := &CSeq{Method: method, Number: cSeqNumber}
	requestLine := &RequestLine{Method: method, RequestUri: d.remoteTarget, SipVersion: SipVersion}

	request := NewRequest()
	request.line = requestLine
	request.SetHeader(d.listeningPoint.CreateViaHeader())
	request.SetHeader(toHeader)
	request.SetHeader(fromHeader)
	request.SetHeader(&callIdHeader)
	request.SetHeader(cSeqHeader)
	request.SetHeader(defaultMaxForwardsHeader.Clone())

	if d.sipStack.Options.UserAgent != "" {
		agent := UserAgent(d.sipStack.Options.UserAgent)
		request.SetHeader(&agent)
	}

	return request
}

func (d *Dialog) CreateRequest(method string) (*Request, error) {
	if d.state == dialogStateTerminated {
		return nil, fmt.Errorf("dialog has terminated")
	}
	if ACK == method || CANCEL == method {
		return nil, fmt.Errorf("disable creat %s requests", method)
	}
	if d.localSeqNumber == nil {
		d.localSeqNumber = newSeqNumber(0)
	}

	d.localSeqNumber.increase()
	return d.createRequest(method, int(*d.localSeqNumber)), nil
}

func (d *Dialog) GetDialogId() string {
	return string(d.dialogId)
}

func (d *Dialog) Terminated() {
	d.state = dialogStateTerminated
}

func (d *Dialog) Delete() {
	d.state = dialogStateTerminated
	d.sipStack.removeDialog(d.GetDialogId())
}
