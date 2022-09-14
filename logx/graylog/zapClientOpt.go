package graylog

import (
	"crypto/tls"

	"go.uber.org/zap"
)

//ZapClientOptions client params options
type ZapClientOptions func(c *logOptions)

//ZapWithFields with fixed fields
func ZapWithFields(field []zap.Field) ZapClientOptions {
	return func(c *logOptions) {
		c.FixFields = field
	}
}

//ZapWithConnWriter with conn send, addr eg:"tcp://10.1.12.89:8891"
func ZapWithConnWriter(addr string, reconnectOnMsg bool) ZapClientOptions {
	return func(c *logOptions) {
		c.GELF.addr, c.GELF.net, _ = parseURI(addr)
		c.isGELF = true
		c.GELF.reconnectOnMsg = reconnectOnMsg
	}
}

//ZapWithTCPMsgSplit configure TCP message mode TCPModeLogstash or TCPModeGraylog, default is TCPModeGraylog
func ZapWithTCPMsgSplit(mode string) ZapClientOptions {
	return func(c *logOptions) {
		c.GELF.tcpMsgMode = mode
	}
}

//ZapWithTLSConnWriter TODO with TLS conn writer
func ZapWithTLSConnWriter(netName, addr string, reconnectOnMsg bool, config *tls.Config) ZapClientOptions {
	return func(c *logOptions) {
		c.GELF.addr = addr
		c.GELF.net = netName
		c.GELF.reconnectOnMsg = reconnectOnMsg
		c.GELF.config = config
	}
}

//ZapWithStdoutDisplay 是否控制台打印日志
func ZapWithStdoutDisplay(stdoutDisplay bool) ZapClientOptions {
	return func(c *logOptions) {
		c.stdoutDisplay = stdoutDisplay
	}
}

//ZapWithLogPath 日志文件路径
func ZapWithLogPath(path string) ZapClientOptions {
	return func(c *logOptions) {
		c.Dir = path
	}
}

//ZapWithRotateCompress 本地日志文件滚动时是否压缩
func ZapWithRotateCompress(enbale bool) ZapClientOptions {
	return func(c *logOptions) {
		c.Compress = enbale
	}
}

//ZapWithRotateType 日志分割模式:时间-TimeDivision, 大小-SizeDivision
func ZapWithRotateType(rotateType RotateType) ZapClientOptions {
	return func(c *logOptions) {
		c.Division = rotateType
	}
}

//ZapWithEncryptFields with log para encrypt fields
func ZapWithEncryptFields(encryptFields []string, depth int) ZapClientOptions {
	return func(c *logOptions) {
		c.encryptFields = encryptFields
		c.encryptDepth = depth
	}
}

//ZapWithMD5Encrypt with md5 encrypt, default is md5
func ZapWithMD5Encrypt() ZapClientOptions {
	return func(c *logOptions) {
		c.cryptor = newMD5Crypto()
	}
}

//ZapWithAESEncrypt with aes encrypt
func ZapWithAESEncrypt(key, iv []byte) ZapClientOptions {
	return func(c *logOptions) {
		c.cryptor = newAESCrypto(key, iv)
	}
}

//ZapWithCallDepth 日志接口调用层数
func ZapWithCallDepth(skip int) ZapClientOptions {
	return func(c *logOptions) {
		c.callSkip = skip
	}
}

//ZapWithAtomicLevel 设置ES日志发送级别
// 默认DEBUG，可选:INFO, WARN, ERROR
func ZapWithAtomicLevel(level string) ZapClientOptions {
	return func(c *logOptions) {
		c.atomicLevel.UnmarshalText([]byte(level))
	}
}

//ZapWithLocalLogLevel 设置本地日志文件写入级别
// 默认DEBUG，可选:INFO, WARN, ERROR
func ZapWithLocalLogLevel(level string) ZapClientOptions {
	return func(c *logOptions) {
		c.localLogLevel.UnmarshalText([]byte(level))
	}
}

//ZapWithLogTimeDivisionUnit 日志按时间分割的单元，Day，可选参数: graylog.Day, graylog.Hour, graylog.Minute
func ZapWithLogTimeDivisionUnit(timeDivisionUnit string) ZapClientOptions {
	return func(c *logOptions) {
		c.TimeUnit = timeUnit(timeDivisionUnit)
	}
}

//ZapWithLogTimeDivisionMaxAge 日志按时间分割的最大保存文件个数
func ZapWithLogTimeDivisionMaxAge(maxAge int) ZapClientOptions {
	return func(c *logOptions) {
		c.MaxAge = maxAge
	}
}

//ZapWithLogSizeDivisionMaxBackups 日志按文件大小分割的最大保存文件个数
func ZapWithLogSizeDivisionMaxBackups(maxBackups int) ZapClientOptions {
	return func(c *logOptions) {
		c.MaxBackups = maxBackups
	}
}

//ZapWithLogSizeDivisionMaxSize 日志按文件大小分割时 文件最大保存(MB)
func ZapWithLogSizeDivisionMaxSize(maxSize int) ZapClientOptions {
	return func(c *logOptions) {
		c.MaxSize = maxSize
	}
}
