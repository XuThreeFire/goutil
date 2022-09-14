package graylog

import (
	"crypto/tls"
)

//ClientOptions client params options
type ClientOptions func(c *logClient)

//WithLogType with log type, eg: log type, must be configured
// indicate which module log the message, such as robot, retrysvr, etc.
func WithLogType(logType string) ClientOptions {
	return func(c *logClient) {
		c.logType = logType
	}
}

//WithLogSource with log para host, eg: IP/pid, must be configured
// indicate which machine log the message, use for debug
func WithLogSource(host string) ClientOptions {
	return func(c *logClient) {
		c.sourceInfo = host
	}
}

//WithEncryptFields with log para encrypt fields
func WithEncryptFields(encryptFields []string) ClientOptions {
	return func(c *logClient) {
		c.EncryptFields = encryptFields
	}
}

//WithFileWriter with file writer
func WithFileWriter(fName string) ClientOptions {
	return func(c *logClient) {
		c.writers = append(c.writers, newfileWriter(fName))
	}
}

//WithHttpWriter with http send
func WithHttpWriter(svrUrl string) ClientOptions {
	return func(c *logClient) {
		c.writers = append(c.writers, newHttpWriter(svrUrl))
	}
}

//WithConnWriter with conn send,init WithConnWriter before init WithSendBufChNum
func WithConnWriter(netName, addr string, reconnectOnMsg bool) ClientOptions {
	return func(c *logClient) {
		c.writers = append(c.writers, newConnWriter(netName, addr, reconnectOnMsg))
	}
}

//WithTLSConnWriter with TLS conn writer
func WithTLSConnWriter(netName, addr string, reconnectOnMsg bool, config *tls.Config) ClientOptions {
	return func(c *logClient) {
		c.writers = append(c.writers, newTLSWriter(netName, addr, reconnectOnMsg, config))
	}
}

//WithMD5Encrypt with md5 encrypt, default is md5
func WithMD5Encrypt() ClientOptions {
	return func(c *logClient) {
		c.crypto = newMD5Crypto()
	}
}

//WithAESEncrypt with aes encrypt
func WithAESEncrypt(key, iv []byte) ClientOptions {
	return func(c *logClient) {
		c.crypto = newAESCrypto(key, iv)
	}
}

//WithQueueMaxLen with sendQueueLen para
func WithQueueMaxLen(maxCount int) ClientOptions {
	return func(c *logClient) {
		c.sendQueueLen = maxCount
	}
}

//WithSendTaskNum with sendQueueLen para
func WithSendTaskNum(maxCount int) ClientOptions {
	return func(c *logClient) {
		c.sendTaskNum = maxCount
	}
}
