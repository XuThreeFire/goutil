package graylog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
)

// error2StdErr print error log to stderr
func error2StdErr(format string, a ...interface{}) (n int, err error) {
	var nowTimeStr = time.Now().String()
	var callerInfo = getCallerInfo(1)
	return fmt.Fprintf(os.Stderr, nowTimeStr+callerInfo+format, a...)
}

// getCallerInfo get caller file, line and function short info in string
func getCallerInfo(skip int) string {
	callerInfo := ""
	if pc, file, line, ok := runtime.Caller(skip + 1); ok {
		funcName := runtime.FuncForPC(pc).Name()
		extractFnName := regexp.MustCompile(`^.*\.(.*)$`)
		fnName := extractFnName.ReplaceAllString(funcName, "$1")
		callerInfo = fmt.Sprintf(" [%s:%d>%s] ",
			filepath.Base(file), line, fnName)
	}

	return callerInfo
}
