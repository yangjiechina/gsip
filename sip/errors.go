package sip

import "fmt"

const (
	ErrorTransactionTimeout = 1
	ErrorIOException        = 2
	ErrorRequestTimeout     = 3
)

type UACError struct {
	code int
	err  error
}

func newClientTransactionTimeoutError() *UACError {
	return newUACError(ErrorTransactionTimeout, fmt.Errorf("client transcation timeout"))
}

func newUACIOExceptionError(err error) *UACError {
	return newUACError(ErrorIOException, err)
}

func newRequestTimeoutExceptionError() *UACError {
	return newUACError(ErrorRequestTimeout, fmt.Errorf("request timeout"))
}

func newUACError(code int, err error) *UACError {
	return &UACError{code: code, err: err}
}

func (u *UACError) Error() string {
	return u.err.Error()
}

func (u *UACError) Code() int {
	return u.code
}
