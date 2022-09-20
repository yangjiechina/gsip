package sip

import (
	"fmt"
	"time"
)

type AutoRefresher interface {
	refresh()
	Stop()
}

type registerRefresh struct {
	sipStack *Stack
	timer    *time.Timer
	interval time.Duration
	request  *Request
	handler  func(status bool, err error)
}

func (r *registerRefresh) refresh() {
	r.request.RemoveTransactionTag()
	r.request.From().Tag = GenerateTag()
	r.request.CSeq().Number++

	listeningPoint := r.sipStack.GetListeningPoint(r.request.Via().transport)
	var responseEvent *ResponseEvent
	clientTransaction, err := listeningPoint.NewClientTransaction(r.request)
	if err == nil {
		responseEvent, err = clientTransaction.Execute()
	}

	if err != nil {
		r.handler(false, err)
	} else {
		if responseEvent.Response.GetStatusCode() != OK {
			r.handler(false, fmt.Errorf("authorization failure"))
		} else {
			r.timer.Reset(r.interval)
			r.handler(true, nil)
		}
	}
}

func (r *registerRefresh) Stop() {
	r.timer.Stop()
}

func newRegisterRefresher(sipStack *Stack, request *Request, handler func(bool, error)) AutoRefresher {
	expires := request.Expires()
	if expires == nil {
		return nil
	}

	r := &registerRefresh{sipStack: sipStack, request: request.Clone(), handler: handler}
	e := expires.ToInt()
	//refresh 5 seconds earlier
	if expires.ToInt() > 5 {
		e -= 5
	}
	r.interval = time.Duration(e) * time.Second

	r.timer = time.AfterFunc(r.interval, r.refresh)
	return r
}

type subscribeRefresh struct {
	sipStack *Stack
	request  *Request
	dialog   *Dialog
	timer    *time.Timer
	interval time.Duration
	handler  func(status, terminated bool, err error)
}

func (s *subscribeRefresh) refresh() {
	request, err := s.dialog.CreateRequest(SUBSCRIBE)
	if err != nil {
		s.Stop()
		s.handler(false, true, err)
		return
	}

	expires := s.request.Expires()
	contact := s.request.Contact()
	event := s.request.Event()
	request.SetHeader(expires)
	request.SetHeader(contact)
	request.SetHeader(event)

	if content := s.request.Content(); content != nil {
		request.SetContent(s.request.ContentType(), content)
	}

	clientTransaction, err := s.sipStack.GetListeningPoint(request.Via().transport).NewClientTransaction(request)
	var responseEvent *ResponseEvent
	if err == nil {
		responseEvent, err = clientTransaction.Execute()
	}

	if err != nil {
		s.handler(false, false, err)
		return
	}

	response := responseEvent.Response
	code := response.GetStatusCode()
	if code == CallTransactionDoesNotExist {
	}

	if code == OK {
		s.timer.Reset(s.interval)
		s.handler(true, false, nil)
	} else {
		s.handler(false, code == CallTransactionDoesNotExist, fmt.Errorf("%d %s", code, response.GetReason()))
	}
}

func (s *subscribeRefresh) Stop() {
	s.timer.Stop()
}

func newSubscribeRefresherWithNear(stack *Stack, request *Request, dialog *Dialog, interval time.Duration, handler func(status bool, terminated bool, err error)) AutoRefresher {
	refresh := &subscribeRefresh{sipStack: stack, request: request.Clone(), dialog: dialog, interval: interval, handler: handler}
	refresh.timer = time.AfterFunc(refresh.interval, refresh.refresh)
	return refresh
}
