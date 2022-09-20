package sip

import (
	"fmt"
	"strings"
	"time"
)

type EventListener interface {
	OnRequest(*RequestEvent)
}

// EventInterceptor You can use it for stateless proxy/**
type EventInterceptor interface {
	OnRequest(*Request)
	OnResponse(*Response)
}

type OnFailure func(err *UACError)

type OnSuccess func(event *ResponseEvent)

// Option Sip stack 的全局参数
type Option struct {
	/**
	预计有多少事务用户
	*/
	TUCount int
	/**
	统一配置UAC请求的超时时间，应该大于0小于事务时间.在绝大部的情况下，我们并不需要长时间等待事务超时，才算做请求失败
	单位 seconds
	*/
	RequestTimeout time.Duration

	UserAgent string
}

type Stack struct {
	Listens []*ListeningPoint
	//TLS     *ListeningPoint
	//WS      *ListeningPoint

	EventListener    EventListener
	EventInterceptor EventInterceptor
	Option           Option

	clientTransactions *SafeMap
	serverTransactions *SafeMap
	dialogs            *SafeMap
}

func (stack *Stack) Stop() {
	if stack.Listens != nil {
		for _, listen := range stack.Listens {
			if listen.transport != nil {
				listen.transport.close()
			}
		}
	}

	if stack.clientTransactions != nil {
		stack.clientTransactions.Clear()
	}
	if stack.serverTransactions != nil {
		stack.serverTransactions.Clear()
	}
	if stack.dialogs != nil {
		stack.dialogs.Clear()
	}
}

func (stack *Stack) Start() error {
	for _, listen := range stack.Listens {
		server, err := createServer(listen.Transport, fmt.Sprintf("%s:%d", listen.IP, listen.Port))
		if err != nil {
			stack.Stop()
			return err
		}

		listen.sipStack = stack
		listen.transport = server
		if TCP == listen.Transport {
			listen.tcpSessions = CreateSafeMap(10)
		}
		server.setHandler(listen)
	}

	stack.clientTransactions = CreateSafeMap(1024)
	stack.serverTransactions = CreateSafeMap(1024)
	stack.dialogs = CreateSafeMap(1024)

	return nil
}

func (stack *Stack) Debug() (int, int, int) {
	var client, server, dialog int
	if stack.clientTransactions != nil {
		client = stack.clientTransactions.Size()
	}

	if stack.serverTransactions != nil {
		server = stack.serverTransactions.Size()
	}

	if stack.dialogs != nil {
		dialog = stack.dialogs.Size()
	}

	return client, server, dialog
}

func (stack *Stack) GetListeningPoint(transport string) *ListeningPoint {
	for _, listen := range stack.Listens {
		if strings.ToUpper(listen.Transport) == strings.ToUpper(transport) {
			return listen
		}
	}

	return nil
}

func (stack *Stack) findDialog(id string) (*Dialog, bool) {
	if find, b := stack.dialogs.Find(id); b {
		return find.(*Dialog), true
	}
	return nil, false
}

func (stack *Stack) removeDialog(id string) *Dialog {
	if remove, b := stack.dialogs.Remove(id); b {
		return remove.(*Dialog)
	}
	return nil
}

func (stack *Stack) addDialog(id string, dialog *Dialog) {
	stack.dialogs.Add(id, dialog)
}

func (stack *Stack) findTransaction(id string, isServer bool) (interface{}, bool) {
	if isServer {
		return stack.serverTransactions.Find(id)
	} else {
		return stack.clientTransactions.Find(id)
	}
}

func (stack *Stack) addTransaction(id string, tx interface{}, isServer bool) {
	//fmt.Printf("添加事务:%s:server:%t address:%p\r\n", id, isServer, stack)
	if isServer {
		stack.serverTransactions.Add(id, tx)
	} else {
		stack.clientTransactions.Add(id, tx)
	}
}

func (stack *Stack) removeTransaction(id string, isServer bool) {
	//fmt.Printf("%s 删除事务:%s:server:%t address:%p\r\n", time.Now().Format("2006-01-02T15:04:05"), id, isServer, stack)
	if isServer {
		stack.serverTransactions.Remove(id)
	} else {
		stack.clientTransactions.Remove(id)
	}
}

func (stack *Stack) StartAutoRefreshWithRegister(request *Request, handler func(bool, error)) AutoRefresher {
	if request.GetRequestMethod() != REGISTER {
		panic("invalid request")
	}
	if err := request.CheckHeaders(); err != nil {
		panic(err)
	}
	return newRegisterRefresher(stack, request, handler)
}

func (stack *Stack) StartAutoRefreshWithSubscribe(request *Request, dialog *Dialog, interval time.Duration, handler func(bool, bool, error)) AutoRefresher {
	if request.GetRequestMethod() != SUBSCRIBE {
		panic("invalid request")
	}
	if err := request.CheckHeaders(); err != nil {
		panic(err)
	}
	return newSubscribeRefresherWithNear(stack, request, dialog, interval, handler)
}
