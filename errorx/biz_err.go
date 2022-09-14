package errorutil

import (
	"github.com/pkg/errors"
)

const (
	SuccessCode        = 100
	SuccessMsg         = "Success"
	ReceiveSuccessCode = 200
	ReceiveSuccessMsg  = "ReceiveSuccess"
)

// BizErr is a definition of error

var errMap = map[error]int32{
	// TODO realize your errorMay
}

// SetErrMap set you errorMap
func SetErrMap(e map[error]int32) {
	errMap = e
}

// ErrCode 匹配 错误码
func ErrCode(err error) (int32, string) {
	if len(errMap) == 0 {
		panic("use SetErrMap to realize your err map!")
	}
	var code int32
	var msg string

	if err != nil {
		oriErr := errors.Cause(err)
		if defCode, ok := errMap[oriErr]; ok {
			code = defCode
		} else {
			code = UnknownCode
		}
		msg = err.Error()
	} else {
		code = SuccessCode
		msg = SuccessMsg
	}

	return code, msg
}
