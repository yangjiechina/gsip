package sip

import "testing"

type MyIntercept struct {
}

func (m *MyIntercept) OnRequest(event *RequestEvent) {
	panic("implement me")
}

func (m *MyIntercept) OnResponse(event *ResponseEvent) {
	panic("implement me")
}

func (m *MyIntercept) OnTxTimeout(transaction Transaction) {
	panic("implement me")
}

func (m *MyIntercept) OnTxTerminate(transaction Transaction) {
	panic("implement me")
}

func (m *MyIntercept) OnDialogTerminate(transaction Transaction) {
	panic("implement me")
}

func (m *MyIntercept) OnIOException(transaction Transaction) {
	panic("implement me")
}

func TestStack(t *testing.T) {
	intercept := &MyIntercept{}
	udpPoint := &ListeningPoint{IP: "0.0.0.0", Port: 5070, Transport: UDP}
	tcpPoint := &ListeningPoint{IP: "0.0.0.0", Port: 5070, Transport: TCP}

	stack := Stack{Listens: []*ListeningPoint{udpPoint, tcpPoint}, EventListener: intercept}
	stack.Start()
	requestUri := &SipUri{
		User: "34020000002000000002",
		HostPort: HostPort{
			Host: "192.168.1.110",
			Port: 5060,
		},
	}

	fromHeader := &From{
		Address: &Address{Uri: &SipUri{
			User: "34020111002000011111",
			HostPort: HostPort{
				Host: "3402011100",
			},
		},
		},
		Tag: GenerateTag(),
	}

	toHeader := &To{
		Address: &Address{Uri: &SipUri{
			User: "34020111002000011111",
			HostPort: HostPort{
				Host: "3402011100",
			},
		},
		},
	}

	registerRequest := udpPoint.NewEmptyRequestMessage(REGISTER, requestUri, fromHeader, toHeader)
	//if err := stack.SendRequestSync(registerRequest, func(event *ResponseEvent) {
	//	println(event)
	//}, func(err error, ioException, txTimeout, requestTimeout bool) {
	//	println(err)
	//}); err != nil {
	//	panic(err)
	//}
	if transaction, err := udpPoint.NewClientTransaction(registerRequest); err != nil {
		transaction.Execute()
	}
}
