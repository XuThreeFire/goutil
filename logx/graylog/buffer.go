package graylog

import (
	"bytes"
	"time"
)

func buffer(b *bytes.Buffer, bch chan *bytes.Buffer) {
	//flush
	b.Reset()
	timer := time.After(20 * time.Millisecond)
sync:
	for i := 0; i < defaultSendBufBatchNum; i++ {
		select {
		case buf := <-bch:
			b.Write(buf.Bytes())
			//b.Write(buf)
			//every 100 ms write sync
		case <-timer:
			break sync
		}
	}
}
