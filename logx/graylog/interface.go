package graylog

func Debug(msg string, args ...interface{}) {
	if loggerSugar == nil {
		return
	}

	loggerSugar.Debugf(msg, args...)
}

func Info(msg string, args ...interface{}) {
	if loggerSugar == nil {
		return
	}

	loggerSugar.Infof(msg, args...)
}

func Error(msg string, args ...interface{}) {
	if loggerSugar == nil {
		return
	}

	loggerSugar.Errorf(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	if loggerSugar == nil {
		return
	}

	loggerSugar.Warnf(msg, args...)
}

func Critical(msg string, args ...interface{}) {
	if loggerSugar == nil {
		return
	}

	loggerSugar.Fatalf(msg, args...)
}

//-------以下为带secondfacility 的日志记录接口----//

func InfoEx(secondfacility string, tag string, elapsedtime int64, msg string, args ...interface{}) {
	if graylogCli == nil {
		return
	}

	graylogCli.log(logINFO, tag, secondfacility, elapsedtime, msg, args...)
}

func DebugEx(secondfacility string, tag string, elapsedtime int64, msg string, args ...interface{}) {
	if graylogCli == nil {
		return
	}
	graylogCli.log(logDEBUG, tag, secondfacility, elapsedtime, msg, args...)
}

func ErrorEx(secondfacility string, tag string, elapsedtime int64, msg string, args ...interface{}) {
	if graylogCli == nil {
		return
	}
	graylogCli.log(logERROR, tag, secondfacility, elapsedtime, msg, args...)
}

func TraceEx(secondfacility string, tag string, elapsedtime int64, msg string, args ...interface{}) {
	if graylogCli == nil {
		return
	}
	graylogCli.log(logTRACE, tag, secondfacility, elapsedtime, msg, args...)
}

func FatalEx(secondfacility string, tag string, elapsedtime int64, msg string, args ...interface{}) {
	if graylogCli == nil {
		return
	}
	graylogCli.log(logFATAL, tag, secondfacility, elapsedtime, msg, args...)
}

//func MonitorEx(secondfacility string, tag string, elapsedtime int64, msg string, args ...interface{}) {
//	if graylogCli == nil {
//		return
//	}
//	graylogCli.log(tag, "Monitor", secondfacility, elapsedtime, msg,  args...)
//}
