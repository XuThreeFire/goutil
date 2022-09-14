package errorutil

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// UnknownCode is unknown code for errorx info.
	UnknownCode = 107
	// UnknownReason is unknown reason for errorx info.
	UnknownReason = "内部操作失败"
)

func (e *Error) Error() string {
	// return fmt.Sprintf("errorx: statusCode = %d statusReason = %s resultStatus = %t", e.StatusCode, e.StatusReason, e.ResultStatus)
	return e.StatusReason
}

// AddMsg returns ecode.Error with msg.
func (e *Error) AddMsg(msg string) *Error {
	e.StatusReason += ":" + msg
	return e
}

// GRPCStatus returns the Status represented by se.
func (e *Error) GRPCStatus() *status.Status {
	return status.New(codes.Code(e.StatusCode), e.StatusReason)
}

// New returns an errorx object for the code, message.
func New(code int, reason string, status bool) *Error {
	return &Error{
		StatusCode:   int32(code),
		StatusReason: reason,
		ResultStatus: status,
	}
}

// Code returns the http code for a errorx.
// It supports wrapped errors.
func Code(err error) int {
	if err == nil {
		return 100 //nolint:gomnd
	}
	if se := FromError(err); se != nil {
		return int(se.StatusCode)
	}
	return UnknownCode
}

// Reason returns the reason for a particular errorx.
// It supports wrapped errors.
func Reason(err error) string {
	if se := FromError(err); se != nil {
		return se.StatusReason
	}
	return UnknownReason
}

// FromError try to convert an errorx to *Error.
// It supports wrapped errors.
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	if se := new(Error); errors.As(err, &se) {
		return se
	}
	return New(UnknownCode, err.Error(), false)
}
