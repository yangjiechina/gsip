package sip

import (
	"fmt"
)

type ServerTransaction struct {
	transaction
}

func (t *ServerTransaction) sendProvisionalResponse(response *Response) {
	t.provisionalResponse = response
	t.provisionalResponseBytes = response.ToBytes()
	sendMessage(t.conn, t.provisionalResponseBytes, t)
}

func (t *ServerTransaction) SendResponse(response *Response) {
	if isDialogCreated(response.cSeq.Method) && response.Contact() == nil && t.listeningPoint.contact != nil {
		response.SetHeader(t.listeningPoint.contact)
	}
	if err := response.CheckHeaders(); err != nil {
		panic(err)
	}

	cSeqHeader := response.CSeq()
	toHeader := response.To()
	var dialog *Dialog
	if toHeader.Tag != "" && isDialogCreated(cSeqHeader.Method) {
		if response.GetStatusCode() > 100 && response.GetStatusCode() < 300 {
			id := response.GetDialogId(true)
			if dialog, _ = t.sipStack.findDialog(id); dialog == nil {
				dialog = createDialog(t.sipStack, t.listeningPoint, t.originalRequest, response, true)
				t.dialog = dialog
				t.sipStack.addDialog(id, dialog)
			}
		}
	}

	if response.GetStatusCode() < 200 {
		if !t.isInvite && unInviteServerStateTrying == t.stateMachine.getState() {
			t.stateMachine.setState(unInviteServerStateProceeding)
			t.sendProvisionalResponse(response)
		} else if t.isInvite && inviteServerStateProceeding == t.stateMachine.getState() {
			//They are not sent reliably by the transaction layer (they are not retransmitted by it) and do not cause a change in the state of the server transaction.
			//TU可以发送任意数量的临时应答，并且不会改变状态
			t.sendProvisionalResponse(response)
		}

	} else if response.GetStatusCode() < 300 {
		t.finalResponse = response
		t.finalResponseBytes = response.ToBytes()
		if sendMessage(t.conn, t.finalResponseBytes, t) != nil {
			return
		}
		if !t.isInvite {
			t.stateMachine.setState(unInviteServerStateCompleted)

		} else if inviteServerStateProceeding == t.stateMachine.getState() {
			contactHeader := response.GetHeader(ContactName)
			if contactHeader == nil {
				panic("Contact StrHeader is mandatory for the OK to the INVITE")
			}
			dialog.state = dialogStateConfirmed
			t.stateMachine.setState(inviteServerStateTerminated)
		}

	} else if response.GetStatusCode() < 700 {
		t.finalResponse = response
		t.finalResponseBytes = response.ToBytes()
		if sendMessage(t.conn, t.finalResponseBytes, t) != nil {
			return
		}

		if !t.isInvite {
			t.stateMachine.setState(unInviteServerStateCompleted)
		} else if inviteServerStateProceeding == t.stateMachine.getState() {
			dialogId := response.GetDialogId(true)
			if dialog = t.sipStack.removeDialog(dialogId); dialog != nil {
				dialog.state = dialogStateTerminated
			}
			//非2xx应答，事务还包含一个ACK请求 等待ACK
			t.stateMachine.setState(inviteServerStateCompleted)
		}

	}
}
func (t *ServerTransaction) retransmit() bool {
	if t.isInvite && inviteServerStateCompleted == t.stateMachine.getState() {
		return sendMessage(t.conn, t.finalResponseBytes, t) != nil
	}
	return true
}

func (t *ServerTransaction) ioException(err error) {
	t.ioError <- err
	t.terminated()
}

func (t *ServerTransaction) terminated() {
	t.sipStack.removeTransaction(t.id, true)
	if t.isInvite {
		t.stateMachine.setState(inviteServerStateTerminated)
	} else {
		t.stateMachine.setState(unInviteServerStateTerminated)
	}
}

func (t *ServerTransaction) timeout() {
	t.txTimeout <- true
	t.terminated()
}

func (t *ServerTransaction) filterDialog(request *Request, dialog *Dialog) error {
	switch request.GetRequestMethod() {
	case ACK, BYE, INFO, NOTIFY:
		if dialog == nil {
			//send 481 Call transaction Does Not Exist
			if ACK != request.GetRequestMethod() {
				response := request.CreateResponse(CallTransactionDoesNotExist)
				t.SendResponse(response)
			}

			return fmt.Errorf("the transaction does not exist")
		}
		break
	}

	if dialog != nil {
		if BYE == request.GetRequestMethod() {
			t.sipStack.removeDialog(request.GetDialogId(true))
			dialog.state = dialogStateTerminated
		}

		cSeqHeader := request.CSeq()
		if dialog.remoteSeqNumber == nil {
			dialog.remoteSeqNumber = newSeqNumber(cSeqHeader.Number)
		} else {
			if dialog.remoteSeqNumber.greater(cSeqHeader.Number) {
				//send 500 Server Internal Error
				response := request.CreateResponse(ServerInternalError)
				t.SendResponse(response)
				return fmt.Errorf("seq number must be incremented")
			}

			dialog.remoteSeqNumber.setValue(cSeqHeader.Number)
		}
	}

	return nil
}

func (t *ServerTransaction) processRequest(request *Request) {
	if t.isInvite {
		if inviteServerStateProceeding > t.stateMachine.getState() {
			t.stateMachine.setState(inviteServerStateProceeding)
			//re-Invite
			//if dialog, _ := t.sipStack.findDialog(request.GetDialogId(true)); dialog != nil {
			//	if header := request.GetHeader(ContactName); header != nil {
			//		header.(*Contact).Address.Uri
			//	}
			//}
			go t.sipStack.EventListener.OnRequest(&RequestEvent{request, nil, t})
		} else if inviteServerStateProceeding == t.stateMachine.getState() && t.provisionalResponseBytes != nil {
			//If a Request retransmission is received while in the "Proceeding" state, the most recent provisional responseEvent that was received from the TU MUST be passed to the transport layer for retransmission.
			sendMessage(t.conn, t.provisionalResponseBytes, t)
		} else if inviteServerStateCompleted == t.stateMachine.getState() {
			//非2XX应答的ACK请求
			if request.GetRequestMethod() == ACK {
				t.stateMachine.setState(inviteServerStateConfirmed)
				go t.sipStack.EventListener.OnRequest(&RequestEvent{request, nil, t})
			} else if t.finalResponseBytes != nil {
				sendMessage(t.conn, t.finalResponseBytes, t)
			}
		}
	} else {
		d, _ := t.sipStack.findDialog(request.GetDialogId(true))
		if err := t.filterDialog(request, d); err != nil {
			return
		}

		if unInviteServerStateTrying > t.stateMachine.getState() {
			t.stateMachine.setState(unInviteServerStateTrying)
			go t.sipStack.EventListener.OnRequest(&RequestEvent{request, d, t})
		} else if unInviteServerStateProceeding == t.stateMachine.getState() && t.provisionalResponseBytes != nil {
			sendMessage(t.conn, t.provisionalResponseBytes, t)
		}
	}
}
