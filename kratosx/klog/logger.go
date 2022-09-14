package klog

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"

	"github.com/XuThreeFire/goutil/logx/graylog"
)

var _ log.Logger = (*GLogger)(nil)

type GLogger struct {
	Logger *zap.Logger
	Sync   func() error
}

// NewGLogger return GLogger
func NewGLogger(addr string) (*GLogger, error) {
	err := graylog.InitZapLog(
		graylog.ZapWithConnWriter(addr, false),
		graylog.ZapWithLogPath("./logs"),
		graylog.ZapWithTCPMsgSplit("logstash"), // 此次使用 logstash收集，所以必须打开
		graylog.ZapWithRotateType(graylog.SizeDivision),
		graylog.ZapWithLogSizeDivisionMaxBackups(30),
		//graylog.ZapWithLogTimeDivisionMaxSize(100),
		graylog.ZapWithAtomicLevel("INFO"), // 上传的level
		// graylog.ZapWithStdoutDisplay(true),
		graylog.ZapWithCallDepth(4),
	)
	if err != nil {
		return nil, err
	}

	return &GLogger{Logger: graylog.Logger, Sync: graylog.Logger.Sync}, nil
}

// NewLLogger return local GLogger
func NewLLogger() (*GLogger, error) {
	// TODO std.logx
	err := graylog.InitZapLog(
		graylog.ZapWithLogPath("./logs"),
		graylog.ZapWithRotateType(graylog.SizeDivision),
		graylog.ZapWithLogSizeDivisionMaxBackups(30),
		//graylog.ZapWithLogTimeDivisionMaxSize(100),
		graylog.ZapWithAtomicLevel("INFO"), // 上传的level
		graylog.ZapWithCallDepth(4),
	)
	if err != nil {
		return nil, err
	}

	return &GLogger{Logger: graylog.Logger, Sync: graylog.Logger.Sync}, nil
}

// Log Implementation of logger interface
func (l *GLogger) Log(level log.Level, keyvals ...interface{}) error {
	if len(keyvals) == 0 || len(keyvals)%2 != 0 {
		l.Logger.Warn(fmt.Sprint("Keyvalues must appear in pairs: ", keyvals))
		return nil
	}

	// Zap.Field is used when keyvals pairs appear
	var data []zap.Field
	for i := 0; i < len(keyvals); i += 2 {
		if keyvals[i] == "msg" {
			keyvals[i] = "message"
		}
		data = append(data, zap.Any(fmt.Sprint(keyvals[i]), fmt.Sprint(keyvals[i+1])))
		//data = append(data, zap.Any(fmt.Sprint(keyvals[i]), keyvals[i+1]))
	}
	switch level {
	case log.LevelDebug:
		l.Logger.Debug("", data...)
	case log.LevelInfo:
		l.Logger.Info("", data...)
	case log.LevelWarn:
		l.Logger.Warn("", data...)
	case log.LevelError:
		l.Logger.Error("", data...)
	}
	return nil
}
