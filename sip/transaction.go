package sip

import (
	"net"
	"sync"
)

const (
	T1 = 500
	T2 = 4000
	T4 = 5000

	inviteClientStateCalling    = 1
	inviteClientStateProceeding = 2
	inviteClientStateCompleted  = 3
	inviteClientStateTerminated = 4

	unInviteClientStateTrying     = 1
	unInviteClientStateProceeding = 2
	unInviteClientStateCompleted  = 3
	unInviteClientStateTerminated = 4

	inviteServerStateProceeding = 1
	inviteServerStateCompleted  = 2
	inviteServerStateConfirmed  = 3
	inviteServerStateTerminated = 4

	unInviteServerStateTrying     = 1
	unInviteServerStateProceeding = 2
	unInviteServerStateCompleted  = 3
	unInviteServerStateTerminated = 4
)

type Transaction interface {
	retransmit() bool // if return false, continue retransmit
	timeout()
	terminated()
	ioException(error)

	GetOriginalRequest() *Request
	GetDialog() *Dialog
}

type transaction struct {
	id                       string
	stateMachine             IStateMachine
	transport                ITransport
	sipStack                 *Stack
	originalRequest          *Request
	originalRequestBytes     []byte
	provisionalResponse      *Response
	provisionalResponseBytes []byte
	finalResponse            *Response
	finalResponseBytes       []byte
	isInvite                 bool
	hop                      *Hop
	conn                     net.Conn
	responseEvent            chan *ResponseEvent
	ioError                  chan error
	txTimeout                chan bool
	//txTerminated             chan bool
	dialog *Dialog
}

func (t *transaction) GetOriginalRequest() *Request {
	return t.originalRequest
}

func (t *transaction) GetDialog() *Dialog {
	return t.dialog
}

func sendMessage(conn net.Conn, data []byte, transaction Transaction) error {
	_, err := conn.Write(data)
	if err != nil {
		transaction.ioException(err)
	}
	return err
}

type IStateMachine interface {
	start()
	stopTimer()
	getState() int
	setState(state int)

	setTransaction(transaction Transaction)
}

type StateMachine struct {
	state       int
	isTcp       bool
	transaction Transaction
}

func (s *StateMachine) setTransaction(transaction Transaction) {
	s.transaction = transaction
}

func (s *StateMachine) getState() int {
	return s.state
}

type InviteClientStateMachine struct {
	StateMachine
	timerA *timerA
	timerB *timerB
	timerD *timerD
}

func (ic *InviteClientStateMachine) setState(state int) {
	ic.state = state

	if state == inviteClientStateCalling {
		if !ic.isTcp {
			ic.timerA = &timerA{}
			ic.timerA.start(ic.transaction.retransmit)
		}
		ic.timerB = &timerB{}
		ic.timerB.start(ic.transaction.timeout)

	} else if state == inviteClientStateProceeding {
		if ic.timerA != nil {
			ic.timerA.stop()
		}

	} else if state == inviteClientStateCompleted {
		if ic.timerA != nil {
			ic.timerA.stop()
		}

		if ic.isTcp {
			ic.transaction.terminated()
		} else {
			ic.timerD = &timerD{}
			ic.timerD.start(ic.transaction.terminated)
		}
	} else if state == inviteClientStateTerminated {
		ic.stopTimer()
	}
}

func (ic *InviteClientStateMachine) stopTimer() {
	if ic.timerA != nil {
		ic.timerA.stop()
	}
	if ic.timerB != nil {
		ic.timerB.stop()
	}
	if ic.timerD != nil {
		ic.timerD.stop()
	}
}

func (ic *InviteClientStateMachine) start() {
	ic.setState(inviteClientStateCalling)
}

type UnInviteClientStateMachine struct {
	StateMachine
	timerE *timerE
	timerF *timerF
	timerK *timerK
	mutex  sync.Mutex
}

func (ic *UnInviteClientStateMachine) setState2(state int) {
	ic.state = state

	if unInviteClientStateTrying == state {
		if !ic.isTcp {
			ic.timerE = &timerE{}
			ic.timerE.start(ic.transaction.retransmit)
		}

		ic.timerF = &timerF{}
		ic.timerF.start(ic.transaction.timeout)
	} else if unInviteClientStateProceeding == state {
		if ic.timerE != nil {
			ic.timerE.setToT2()
		}
	} else if unInviteClientStateCompleted == state {
		if ic.timerE != nil {
			ic.timerE.stop()
			ic.timerE = nil
		}
		if ic.timerF != nil {
			ic.timerF.stop()
			ic.timerF = nil
		}

		if ic.isTcp {
			ic.transaction.terminated()
		} else {
			ic.timerK = &timerK{}
			ic.timerK.start(ic.transaction.terminated)
		}
	} else if unInviteClientStateTerminated == state {
		ic.stopTimer()
	}
}
func (ic *UnInviteClientStateMachine) setState(state int) {
	ic.mutex.Lock()
	defer ic.mutex.Unlock()
	ic.setState2(state)
}

func (ic *UnInviteClientStateMachine) stopTimer() {
	if ic.timerE != nil {
		ic.timerE.stop()
		ic.timerE = nil
	}
	if ic.timerF != nil {
		ic.timerF.stop()
		ic.timerF = nil
	}
	if ic.timerK != nil {
		ic.timerK.stop()
		ic.timerK = nil
	}
}

func (ic *UnInviteClientStateMachine) start() {
	ic.setState(unInviteClientStateTrying)
}

type InviteServerStateMachine struct {
	StateMachine
	timerG *timerG
	timerH *timerH
	timerI *timerI
}

func (i *InviteServerStateMachine) setState(state int) {
	i.state = state
	if inviteServerStateProceeding == state {

	} else if inviteServerStateCompleted == state {
		if !i.isTcp {
			//start timer G T1开始 翻倍递增 MIN(2*T1,T2)
			i.timerG = &timerG{}
			i.timerG.start(i.transaction.retransmit)
		}
		i.timerH = &timerH{}
		i.timerH.start(i.transaction.timeout)
	} else if inviteServerStateConfirmed == state {
		if i.timerG != nil {
			i.timerG.stop()
		}
		if i.timerH != nil {
			i.timerH.stop()
		}

		if i.isTcp {
			i.timerI = &timerI{}
			i.timerI.start(i.transaction.terminated)
		} else {
			i.transaction.terminated()
		}
	} else if inviteServerStateTerminated == state {
		i.stopTimer()
	}
}

func (i *InviteServerStateMachine) stopTimer() {
	if i.timerG != nil {
		i.timerG.stop()
	}
	if i.timerH != nil {
		i.timerH.stop()
	}
	if i.timerI != nil {
		i.timerI.stop()
	}
}

func (i *InviteServerStateMachine) start() {
	i.setState(inviteServerStateProceeding)
}

type UnInviteServerStateMachine struct {
	StateMachine
	timerJ *timerJ
}

func (u *UnInviteServerStateMachine) setState(state int) {
	u.state = state
	if unInviteServerStateTrying == state {

	} else if unInviteServerStateProceeding == state {

	} else if unInviteServerStateCompleted == state {
		if !u.isTcp {
			u.timerJ = &timerJ{}
			u.timerJ.start(u.transaction.terminated)
		} else {
			u.transaction.terminated()
		}
	} else if unInviteServerStateTerminated == state {
		u.stopTimer()
	}
}

func (u *UnInviteServerStateMachine) stopTimer() {
	if u.timerJ != nil {
		u.timerJ.stop()
	}
}

func (u *UnInviteServerStateMachine) start() {
	u.setState(unInviteServerStateTrying)
}
