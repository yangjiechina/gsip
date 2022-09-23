package sip

import (
	"context"
)

type ClientTransaction struct {
	transaction
	timeoutCtx context.Context
}

func isDialogCreated(method string) bool {
	switch method {
	case INVITE, SUBSCRIBE, REFER:
		return true
	default:
		return false
	}
}

func (t *ClientTransaction) createAck(response *Response) *Request {
	callIdHeader := t.originalRequest.CallID()
	fromHeader := t.originalRequest.From()
	requestLine := t.originalRequest.GetRequestLine()
	viaHeader := t.originalRequest.Via()
	cSeqHeader := t.originalRequest.CSeq()
	cSeqHeader.Method = ACK

	toHeader := response.To()

	request := NewRequest()
	request.line = &RequestLine{ACK, requestLine.RequestUri.Clone(), SipVersion}
	request.SetHeader(callIdHeader.Clone())
	request.SetHeader(fromHeader.Clone())
	request.SetHeader(viaHeader.Clone())
	request.SetHeader(cSeqHeader.Clone())
	request.SetHeader(toHeader.Clone())

	return request
}

func (t *ClientTransaction) emit(response *Response, dialog *Dialog) {
	//if response.GetStatusCode() < 200 {
	//	t.provisionalResponse = response
	//} else {
	//	t.finalResponse = response
	//}
	t.responseEvent <- &ResponseEvent{response, dialog, t}
}

func (t *ClientTransaction) processResponse(response *Response) {
	code := response.GetStatusCode()
	toHeader := response.To()
	state := t.stateMachine.getState()
	cSeqHeader := response.CSeq()

	var dialog *Dialog
	if toHeader.Tag != "" && isDialogCreated(cSeqHeader.Method) && code >= 100 && code < 300 {
		id := response.GetDialogId(false)
		if code >= 200 {
			if header := response.GetHeader(ContactName); header == nil {
				t.sipStack.removeDialog(id)
				println("The Contact Header was not found in the response")
				return
			}
		}
		if dialog, _ = t.sipStack.findDialog(id); dialog == nil {
			dialog = createDialog(t.sipStack, t.originalRequest, response, false)
			t.dialog = dialog
			t.sipStack.addDialog(id, dialog)
		}
	} else if code == CallTransactionDoesNotExist {
		if removeDialog := t.sipStack.removeDialog(response.GetDialogId(false)); removeDialog != nil {
			removeDialog.state = dialogStateTerminated
		}
	}

	if t.isInvite {
		if code < 200 && state <= inviteClientStateProceeding {
			t.stateMachine.setState(inviteClientStateProceeding)
			if dialog != nil {
				//create early Dialog
				dialog.state = dialogStateEarly
			}
			t.emit(response, dialog)
		} else if code >= 300 {
			if state <= inviteClientStateProceeding {
				t.stateMachine.setState(inviteClientStateCompleted)
				t.sipStack.removeDialog(response.GetDialogId(false))
				t.responseEvent <- &ResponseEvent{response, nil, t}
			}
			//非2XX应答，事务还包含一个ACK请求，每一个重发的响应后发送ACK
			//2XX应答，ACK是一个单独的事务，由TU自己发
			ack := t.createAck(response)
			sendMessage(t.conn, ack.ToBytes(), t)
		} else if code/2 == 100 && state <= inviteClientStateProceeding {
			t.stateMachine.setState(inviteClientStateTerminated)
			if dialog != nil {
				dialog.state = dialogStateConfirmed
			}

			t.emit(response, dialog)
		}
	} else {

		if code < 200 && state == unInviteClientStateTrying {
			t.stateMachine.setState(unInviteClientStateProceeding)
			if dialog != nil {
				dialog.state = dialogStateConfirmed
			}
			t.emit(response, dialog)
		} else if code >= 200 && (state <= unInviteClientStateProceeding) {
			t.stateMachine.setState(unInviteClientStateCompleted)
			t.emit(response, dialog)
		}
	}
}

func (t *ClientTransaction) retransmit() bool {
	if (t.isInvite && t.stateMachine.getState() == inviteClientStateCalling) ||
		(!t.isInvite && t.stateMachine.getState() <= unInviteClientStateProceeding) {
		return sendMessage(t.conn, t.originalRequestBytes, t) != nil
	}

	return true
}

func (t *ClientTransaction) ioException(err error) {
	t.ioError <- err
	t.terminated()
}

func (t *ClientTransaction) terminated() {
	t.sipStack.removeTransaction(t.id, false)
	//t.txTerminated <- true
	if t.isInvite {
		t.stateMachine.setState(inviteClientStateTerminated)
	} else {
		t.stateMachine.setState(unInviteClientStateTerminated)
	}
}

func (t *ClientTransaction) timeout() {
	t.txTimeout <- true
	t.terminated()
}

func (t *ClientTransaction) Execute() (response *ResponseEvent, err error) {
	var evt *ResponseEvent
	var e error
	t.SendRequest(func(event *ResponseEvent) {
		evt = event
	}, func(err *UACError) {
		e = err
	})

	return evt, e
}

func (t *ClientTransaction) SendRequest(onSuccess OnSuccess, onFailure OnFailure) {
	if isDialogCreated(t.originalRequest.cSeq.Method) && t.originalRequest.Contact() == nil && t.listeningPoint.contact != nil {
		t.originalRequest.SetHeader(t.listeningPoint.contact)
	}
	if err := t.originalRequest.CheckHeaders(); err != nil {
		panic(err)
	}

	conn, err := t.sipStack.GetListeningPoint(t.originalRequest.GetTransport()).getConn(t.hop)
	if conn != nil {
		t.conn = conn
		t.originalRequestBytes = t.originalRequest.ToBytes()
		err = sendMessage(t.conn, t.originalRequestBytes, t)
	}

	if err != nil {
		t.terminated()
		if onFailure != nil {
			onFailure(newUACIOExceptionError(err))
		}
		return
	}

	t.stateMachine.start()
	if onSuccess != nil || onFailure != nil {

		var wait func()

		if t.timeoutCtx == nil {
			wait = func() {
				for {
					select {
					case responseEvt := <-t.responseEvent:
						//如果是临时响应,说明有多个响应包 才用协程回调
						if responseEvt.Response.GetStatusCode() < 200 {
							go onSuccess(responseEvt)
						} else {
							onSuccess(responseEvt)
							return
						}
						break
					case exception := <-t.ioError:
						onFailure(newUACIOExceptionError(exception))
						return
					case <-t.txTimeout:
						onFailure(newClientTransactionTimeoutError())
						return
					}
				}
			}

		} else {
			wait = func() {
				for {
					select {
					case responseEvt := <-t.responseEvent:
						//如果是临时响应,说明有多个响应包 才用协程回调
						if responseEvt.Response.GetStatusCode() < 200 {
							go onSuccess(responseEvt)
						} else {
							onSuccess(responseEvt)
						}
						return
					case exception := <-t.ioError:
						onFailure(newUACIOExceptionError(exception))
						return
					case <-t.txTimeout:
						onFailure(newClientTransactionTimeoutError())
						return
					case <-t.timeoutCtx.Done():
						t.terminated()
						onFailure(newRequestTimeoutExceptionError())
						return
					}
				}
			}
		}

		wait()
	}
}
