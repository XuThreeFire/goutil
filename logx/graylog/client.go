package graylog

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	defaultSendQueueLen    = 10000
	defaultSendThreads     = 20
	defaultSendBufChNum    = 0x2800 //allow 10240 rows log
	defaultSendBufBatchNum = 10     //everytime pkg 10 to send

	defaultRetryWaitMs           = 500
	defaultMaxRetry              = 13
	defaultMaxRetryWaitMs        = 60000
	defaultReconnectWaitIncrRate = 1.5

	defaultLogSize = 0x400 //one time log cost 1kb
	logtimeFmt     = "2006-01-02 15:04:05.999 -07:00"
)

const (
	logDEBUG = "debug"
	logINFO  = "info"
	logTRACE = "trace"
	logERROR = "error"
	logFATAL = "fatal"
)

var graylogCli *logClient

type logClient struct {
	sendQueue chan *logData

	sendQueueLen int
	sendTaskNum  int
	sendBufChNum int
	sourceInfo   string //"IP/pid"
	logType      string

	// enc fields
	EncryptFields []string

	// crypto(md5/aes)
	crypto cryptoInterface

	// writers(file/http/tcp/udp)
	writers []writerInterface
}

func (c *logClient) buildLog(level, tag, value, secondfacility string, elapsedtime int64) (data *logData) {
	data = new(logData)
	data.Source = c.sourceInfo
	data.Facility = c.logType

	data.Tag = tag
	data.ElapsedTime = elapsedtime
	data.Message = c.encryptContent(value)
	data.Level = level
	data.LogTime = time.Now().Format(logtimeFmt)
	data.SecondFacility = secondfacility

	return data
}

func (c *logClient) log(level, tag, secondfacility string, elapsedtime int64, msg string, args ...interface{}) {
	value := fmt.Sprintf(msg, args...)
	data := c.buildLog(level, tag, value, secondfacility, elapsedtime)

	select {
	case c.sendQueue <- data:
	case <-time.After(10 * time.Millisecond):
	}
}

func NewGraylogClient(opts ...ClientOptions) error {
	if graylogCli != nil {
		return nil
	}
	graylogCli = &logClient{sendQueueLen: defaultSendQueueLen, sendTaskNum: defaultSendThreads}
	return graylogCli.init(opts...)
}

func (c *logClient) init(opts ...ClientOptions) error {
	c.sendQueue = make(chan *logData, c.sendQueueLen)

	thread := make(chan struct{}, c.sendTaskNum)

	for _, opt := range opts {
		opt(c)
	}

	if c.logType == "" {
		return errors.New("logType must be configured")
	}
	if c.sourceInfo == "" {
		return errors.New("logSource must be configured")
	}
	if len(c.writers) <= 0 {
		return errors.New("at least 1 writer must be configured")
	}

	go func() {
		for data := range c.sendQueue {
			thread <- struct{}{}
			//			length := len(c.sendQueue)
			go func(d *logData) {
				defer func() { <-thread }()
				//	d.ElapsedTime = fmt.Sprintf("%d", l)
				c.sendOut(d)
			}(data)
		}
	}()
	return nil
}

// close the sendQueue chan(s)
func (c *logClient) close() {
	for {
		select {
		case <-time.After(10 * time.Millisecond):
			if len(c.sendQueue) <= 0 {
				close(c.sendQueue)
				return
			}
		}
	}
}

// send log
func (c *logClient) sendOut(data *logData) {
	bytes, _ := json.Marshal(data)
	for _, w := range c.writers {
		_, err := w.Write(bytes)
		if err != nil {
			select {
			case c.sendQueue <- data:
			case <-time.After(10 * time.Millisecond):
			}
		}
	}
}

// encrypt log content
func (c *logClient) encryptContent(content string) string {
	for _, rule := range c.EncryptFields {
		reg := regexp.MustCompile(`"` + rule + `":"[^"]*"`)
		content = reg.ReplaceAllStringFunc(content, c.encField)
	}
	return content
}

// encrypt one filed
func (c *logClient) encField(field string) string {
	var info, infoPre, infoVal, infoAft string
	iEnd := strings.LastIndex(field, `"`)
	if iEnd != -1 {
		info = field[:iEnd]
		infoAft = field[iEnd:]
		iStart := strings.LastIndex(info, `"`)
		if iStart != -1 {
			infoVal = info[iStart+1:]
			infoPre = info[:iStart+1]
		}
	}
	encryptInfo, err := c.crypto.encrypt([]byte(infoVal))
	if err != nil {
		return field
	}
	return infoPre + string(encryptInfo) + infoAft
}
