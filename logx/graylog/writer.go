package graylog

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// File and directory permissions.
const (
	defaultFilePermissions      = 0666
	defaultDirectoryPermissions = 0767
	defaultChunkSize            = 1120
	maxChunkCount               = 128

	compressionNone = 0
	compressionGzip = 1
)

const (
	_GraylogTCPMsgSplit  = '\x00'
	_LogstashTCPMsgSplit = '\x0a'

	_LogstashMode = "logstash"
	_GraylogMode  = "graylog"
)

// write type const
const (
	writeTypeHTTP = "http"
	writeTypeConn = "conn"
	writeTypeUDP  = "udp"
	writeTypeTCP  = "tcp"
	writeTypeTLS  = "tls"
	writeTypeFile = "file"
)

// writer interface
type writerInterface interface {
	//Write data return write length n and err
	Write(data []byte) (n int, err error)
	// close
	close() error
	// get writeType
	getType() string

	Sync() error

	Stop()
}

// connWriter is used to write to a stream-oriented network connection
type connWriter struct {
	innerWriter           io.WriteCloser
	reconnectOnMsg        bool
	reconnect             bool
	retryReconnectMax     int
	retryReconnectWait    int
	retryReconnectMaxWait int
	net                   string
	addr                  string
	useTLS                bool
	configTLS             *tls.Config
	writeType             string
	chunkDataSize         int
	chunkSize             int
	compressionLevel      int
	compressionType       int

	tcpMsgSpilt byte // graylog 和 logstash tcp分割为 \x00 和 \n

	enableEncrypt bool
	encryptFields []string
	encryptDepth  int
	encInterface  cryptoInterface

	//one log time cost 1024 byte,batch sendBufCh 1000 ->  one time send
	sendBufCh      chan *bytes.Buffer
	sendBufChSize  int
	sendBufChClose chan struct{}
}

// Creates writer to the address addr on the network netName
// Connection will be opened on each write if reconnectOnMsg = true
func newConnWriter(netName, addr string, reconnectOnMsg bool) *connWriter {
	cw := &connWriter{}
	cw.net = netName
	cw.addr = addr
	cw.reconnectOnMsg = reconnectOnMsg
	if strings.Contains(netName, writeTypeUDP) {
		cw.writeType = writeTypeUDP
	} else if strings.Contains(netName, writeTypeTCP) {
		cw.writeType = writeTypeTCP
	} else {
		cw.writeType = writeTypeConn
	}

	cw.chunkDataSize = defaultChunkSize - 12
	cw.chunkSize = defaultChunkSize
	cw.compressionType = compressionGzip
	cw.compressionLevel = gzip.BestCompression

	cw.tcpMsgSpilt = _GraylogTCPMsgSplit

	//async push cfg
	cw.fix()
	cw.sendBufCh = make(chan *bytes.Buffer, cw.sendBufChSize)
	cw.sendBufChClose = make(chan struct{}, 0)
	go cw.writeDaemon()

	return cw
}

// Creates a writer that uses ssl/tls
func newTLSWriter(netName, addr string, reconnectOnMsg bool, config *tls.Config) *connWriter {
	cw := &connWriter{}
	cw.net = netName
	cw.addr = addr
	cw.reconnectOnMsg = reconnectOnMsg
	cw.useTLS = true
	cw.configTLS = config
	cw.writeType = writeTypeTLS
	go cw.writeDaemon()
	return cw
}

func (w *connWriter) fix() {
	// avoid not init
	if w.sendBufChSize == 0 {
		w.sendBufChSize = defaultSendBufChNum
	}

	if w.retryReconnectMax == 0 {
		w.retryReconnectMax = defaultMaxRetry
	}

	if w.retryReconnectWait == 0 {
		w.retryReconnectWait = defaultRetryWaitMs
	}

	if w.retryReconnectMaxWait == 0 {
		w.retryReconnectMaxWait = defaultMaxRetryWaitMs
	}

}

// close
func (w *connWriter) close() error {
	if w.innerWriter != nil {
		return w.innerWriter.Close()
	}
	return nil
}

func (w *connWriter) Stop() {
	w.closeBufCh()
}

func (w *connWriter) closeBufCh() {
	w.sendBufChClose <- struct{}{}
}

// set tcp split tag
func (w *connWriter) setTCPMsgMode(mode string) {
	if mode == _LogstashMode {
		w.tcpMsgSpilt = _LogstashTCPMsgSplit
	}
}

// set encrypt options
func (w *connWriter) setEncOpt(fields []string, depth int, encInterface cryptoInterface) error {
	if len(fields) <= 0 {
		return errors.New("encrypt fields is empty")
	}
	if encInterface == nil {
		return errors.New("encrypt interface is empty")
	}

	w.encryptFields = fields
	w.encInterface = encInterface
	w.encryptDepth = depth
	w.enableEncrypt = true

	return nil
}

// encrypt log json content
func (w *connWriter) encryptContent(content string) string {
	trans := "\\"
	for _, rule := range w.encryptFields {
		for i := 0; i < w.encryptDepth+1; i++ {
			// TODO 优化字符串处理性能
			actTrans := "\""
			idxActTrans := actTrans
			for j := 0; j < i*2; j++ {
				actTrans = trans + actTrans
			}
			for k := 0; k < i; k++ {
				idxActTrans = trans + idxActTrans
			}
			reg := regexp.MustCompile(actTrans +
				rule + actTrans +
				" ?: ?" + actTrans +
				"[^" + actTrans +
				"]*" + actTrans)

			content = reg.ReplaceAllStringFunc(content,
				func(field string) string {
					var info, infoPre, infoVal, infoAft string
					iEnd := strings.LastIndex(field, idxActTrans)
					if iEnd != -1 {
						info = field[:iEnd]
						infoAft = field[iEnd:]
						iStart := strings.LastIndex(info, idxActTrans)
						if iStart != -1 {
							infoVal = info[iStart+len(idxActTrans):]
							infoPre = info[:iStart+len(idxActTrans)]
						}
					}

					// skip when empty
					if infoVal == "" {
						return field
					}

					encryptInfo, err := w.encInterface.encrypt([]byte(infoVal))
					if err != nil {
						return field
					}
					return infoPre + "§" + string(encryptInfo) + "§" + infoAft
				})
		}
	}
	return content
}

// write
func (w *connWriter) Write(data []byte) (n int, err error) {
	// encrypt appointed fields
	if w.enableEncrypt {
		data = []byte(w.encryptContent(string(data)))
	}

	if w.reconnectOnMsg {
		defer w.innerWriter.Close()
	}

	switch w.writeType {
	case writeTypeUDP:
		n, err = w.udpWrite(data)
	case writeTypeTCP:
		buf := &bytes.Buffer{}
		buf.Write(data)
		buf.WriteByte(w.tcpMsgSpilt)
		tcpC, ok := w.innerWriter.(*net.TCPConn)
		if ok {
			tcpC.SetWriteDeadline(time.Now().Add(700 * time.Millisecond))
			tcpC.SetReadDeadline(time.Now().Add(700 * time.Millisecond))
		}
		w.process(buf)
	default:
		buf := &bytes.Buffer{}
		buf.Write(data)
		w.process(buf)
	}

	return n, nil
}

func (w *connWriter) connect() error {
	if w.innerWriter != nil {
		w.innerWriter.Close()
		w.innerWriter = nil
	}

	if w.useTLS {
		dialer := net.Dialer{Timeout: 200 * time.Millisecond}
		conn, err := tls.DialWithDialer(&dialer, w.net, w.addr, w.configTLS)
		if err != nil {
			return err
		}

		w.innerWriter = conn
	}
	conn, err := net.DialTimeout(w.net, w.addr, 200*time.Millisecond)
	if err != nil {
		return err
	}

	tcpConn, ok := conn.(*net.TCPConn)
	if ok {
		tcpConn.SetKeepAlive(true)
	}

	w.innerWriter = conn
	if w.innerWriter == nil {
		return errors.New("connect failed, innerWriter is nil")
	}

	return nil
}

// Write implements io.Writer.
func (w *connWriter) udpWrite(buf []byte) (n int, err error) {
	var (
		cw   io.WriteCloser
		cBuf = bytes.NewBuffer([]byte{})
	)

	switch w.compressionType {
	case compressionNone:
		cw = &writeCloser{cBuf}
	case compressionGzip:
		cw, err = gzip.NewWriterLevel(cBuf, gzip.BestCompression)
		//	case CompressionZlib:
		//		cw, err = zlib.NewWriterLevel(&cBuf, w.compressionLevel)
	}

	if err != nil {
		return 0, err
	}

	if n, err = cw.Write(buf); err != nil {
		return n, err
	}

	cw.Close()

	//cBytes := cBuf.Bytes()
	//put data in sendBufCh use async push
	w.process(cBuf)

	return n, nil
}

func (w *connWriter) process(sendBuf *bytes.Buffer) {
	timer := time.After(30 * time.Millisecond)
	select {
	case w.sendBufCh <- sendBuf:
	case <-timer:
		error2StdErr("set data to channel timeout: %s\n", sendBuf.String())
	default:
	}
}

func (w *connWriter) writeDaemon() {
	// TODO 多协程处理, sync.Pool接管
	bytesBuf := &bytes.Buffer{}

	for {
		w.writeDaemonExec(bytesBuf)
		select {
		case <-w.sendBufChClose:
			return
		default:
		}
	}
}

func (w *connWriter) writeDaemonExec(bytesBuf *bytes.Buffer) {
	var err error

	if bytesBuf == nil {
		return
	}

	buffer(bytesBuf, w.sendBufCh)
	if bytesBuf.Len() == 0 {
		return
	}

	if w.net == writeTypeUDP {
		udpC, ok := w.innerWriter.(*net.UDPConn)
		if ok {
			udpC.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
			udpC.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		} else {
			error2StdErr("unexcepted writer, not UDP writer\n")
			return
		}

		if count := w.chunkCount(bytesBuf.Bytes()); count > 1 {
			_, err = w.writeChunked(count, bytesBuf.Bytes())
			if err != nil {
				error2StdErr("writed udp bytes err:%v\n", err)
				return
			}
		}
		return
	}

	n, err := w.safeWrite(bytesBuf.Bytes())
	if err != nil {
		error2StdErr("writed tcp bytes err: %v\n", err)
		w.innerWriter = nil
		return
	}
	if n != bytesBuf.Len() {
		error2StdErr("writed %d bytes but should %d bytes\n", n, bytesBuf.Len())
		return
	}
}

func (w *connWriter) safeWrite(data []byte) (int, error) {
	//note: test retry connect
	//w.innerWriter = nil
	if w.innerWriter == nil {
		error2StdErr("connect retry err: w.innerWriter is nil")
		err := w.connectWithRetry()
		if err != nil {
			error2StdErr("connect retry err:%v\n", err)
			return 0, err
		}
	}

	return w.innerWriter.Write(data)
}

func (w *connWriter) connectWithRetry() (err error) {
	timeout := time.NewTimer(time.Duration(0))
	defer func() {
		timeout.Stop()
	}()
	for i := 0; i < w.retryReconnectMax; i++ {
		select {
		case <-timeout.C:
			err = w.connect()
			if err == nil {
				return nil
			}
			if err != nil {
				return err
			}
			waitTime := w.retryReconnectWait * pow(defaultReconnectWaitIncrRate, float64(i-1))
			if waitTime > w.retryReconnectMaxWait {
				waitTime = w.retryReconnectMaxWait
			}
			timeout = time.NewTimer(time.Duration(waitTime) * time.Millisecond)
		}
	}

	return fmt.Errorf("could not connect to %s after %d retries", w.net, w.retryReconnectMax)
}

// chunkCount calculate the number of GELF chunks.
func (w *connWriter) chunkCount(b []byte) int {
	lenB := len(b)
	if lenB <= w.chunkSize {
		return 1
	}

	return len(b)/w.chunkDataSize + 1
}

// writeChunked send message by chunks.
func (w *connWriter) writeChunked(count int, cBytes []byte) (n int, err error) {
	if count > maxChunkCount {
		return 0, fmt.Errorf("need %d chunks but shold be later or equal to %d",
			count, maxChunkCount)
	}

	var (
		cBuf = bytes.NewBuffer(
			make([]byte, 0, w.chunkSize),
		)
		nChunks   = uint8(count)
		messageID = make([]byte, 8)
	)

	if n, err = io.ReadFull(rand.Reader, messageID); err != nil || n != 8 {
		return 0, fmt.Errorf("rand.Reader: %d/%s", n, err)
	}

	var (
		off       int
		chunkLen  int
		bytesLeft = len(cBytes)
	)

	for i := uint8(0); i < nChunks; i++ {
		off = int(i) * w.chunkDataSize
		chunkLen = w.chunkDataSize
		if chunkLen > bytesLeft {
			chunkLen = bytesLeft
		}

		cBuf.Reset()
		cBuf.Write([]byte{0x1e, 0x0f})
		cBuf.Write(messageID)
		cBuf.WriteByte(i)
		cBuf.WriteByte(nChunks)
		cBuf.Write(cBytes[off : off+chunkLen])

		if n, err = w.safeWrite(cBuf.Bytes()); err != nil {
			return len(cBytes) - bytesLeft + n, fmt.Errorf("udp:: write failed,err:%v", err)
		}

		if n != len(cBuf.Bytes()) {
			n = len(cBytes) - bytesLeft + n
			return n, fmt.Errorf("writed %d bytes but should %d bytes", n, len(cBytes))
		}

		bytesLeft -= chunkLen
	}

	if bytesLeft != 0 {
		return len(cBytes) - bytesLeft, fmt.Errorf("error: %d bytes left after sending", bytesLeft)
	}

	return len(cBytes), nil
}

func (w *connWriter) Sync() error {
	return nil
}

func (w *connWriter) neededConnectOnMsg() bool {
	if w.reconnect {
		w.reconnect = false
		return true
	}

	if w.innerWriter == nil {
		return true
	}

	return w.reconnectOnMsg
}

func (w *connWriter) getType() string {
	return w.writeType
}

// http writer
type httpWriter struct {
	// url
	serverUrl string
	writeType string

	sendBufCh      chan []byte
	sendBufChSize  int
	sendBufChClose chan struct{}
}

// new httpWriter
func newHttpWriter(svrUrl string) *httpWriter {
	http.DefaultTransport.(*http.Transport).MaxConnsPerHost = 20
	// http.DefaultClient.Timeout = 30 * time.Second
	hw := &httpWriter{serverUrl: svrUrl, writeType: writeTypeHTTP}
	hw.sendBufCh = make(chan []byte, defaultSendBufChNum)
	hw.sendBufChClose = make(chan struct{})
	go hw.writeDaemon()
	return hw
}

// http close
func (hw *httpWriter) close() error {
	return nil
}

func (hw *httpWriter) closeBufCh() {
	hw.sendBufChClose <- struct{}{}
}

func (hw *httpWriter) Stop() {
	hw.sendBufChClose <- struct{}{}
}

// http write
func (hw *httpWriter) Write(data []byte) (n int, err error) {
	hw.process(data)
	return
}

func (hw *httpWriter) process(data []byte) {
	timer := time.After(30 * time.Millisecond)
	select {
	case hw.sendBufCh <- data:
	case <-timer:
		error2StdErr("set data to channel timeout, data length: %d\n", len(data))
	default:
	}
}

func (hw *httpWriter) writeDaemon() {
	bytesBuf := &bytes.Buffer{}

	for {
		hw.send(bytesBuf.Bytes())
		select {
		case <-hw.sendBufChClose:
			return
		default:
		}
	}
}

func (hw *httpWriter) send(data []byte) {
	reader := bytes.NewReader(data)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", hw.serverUrl, reader)
	if err != nil {
		error2StdErr("send data new request failed:%v\n", err)
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		error2StdErr("send data do http failed:%v\n", err)
	}
	defer res.Body.Close()
	// GELF http 发送返回的http code 202,内容为空
	if res.StatusCode != http.StatusAccepted {
		error2StdErr("send data response with http code %d\n ", res.StatusCode)
	}
}

func (hw *httpWriter) Sync() error {
	return nil
}

// get type
func (hw *httpWriter) getType() string {
	return hw.writeType
}

// file writer TODO:https://github.com/google/vectorio,利用writev 一次性刷,减少syscall同时减少memcpy
type fileWriter struct {
	fileName    string
	innerWriter io.WriteCloser
	writeType   string
}

const (
	defaultFileName = "./logs/log"
)

var defaultFileWriter = fileWriter{
	fileName: defaultFileName,
}

// new fileWriter
func newfileWriter(fname string) *fileWriter {
	return &fileWriter{fileName: fname, writeType: writeTypeFile}
}

// close
func (fw *fileWriter) close() error {
	if fw.innerWriter != nil {
		err := fw.innerWriter.Close()
		if err != nil {
			return err
		}

		fw.innerWriter = nil
	}
	return nil
}

// create file
func (fw *fileWriter) createFile() error {
	folder, _ := filepath.Split(fw.fileName)
	var err error

	if 0 != len(folder) {
		err = os.MkdirAll(folder, defaultDirectoryPermissions)
		if err != nil {
			return err
		}
	}

	// If exists
	fw.innerWriter, err = os.OpenFile(fw.fileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, defaultFilePermissions)

	if err != nil {
		return err
	}

	return nil
}

func (fw *fileWriter) Sync() error {
	return nil
}

func (fw *fileWriter) Stop() {
	fw.close()
}

// file write
func (fw *fileWriter) Write(data []byte) (n int, err error) {
	if fw.innerWriter == nil {
		if err := fw.createFile(); err != nil {
			return 0, err
		}
	}
	return fw.innerWriter.Write(data)
}

// get type
func (fw *fileWriter) getType() string {
	return fw.writeType
}

func parseURI(uri string) (addr, scheme string, err error) {
	var p *url.URL
	p, err = url.Parse(uri)
	if err != nil {
		return
	}
	return p.Host, p.Scheme, nil
}

type writeCloser struct {
	*bytes.Buffer
}

// Close implementation of io.WriteCloser.
func (*writeCloser) Close() error {
	return nil
}
