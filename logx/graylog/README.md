# graylog 客户端

-   使用 graylog的 GELF 格式
-   发送支持 HTTP/TCP/UDP/TLS
-   支持使用zap日志库进行日志记录本地+远端（UDP）
-   本地日志文件默认为按大小切割，且进行归档gzip压缩，备份15份，每个文件100MB

## 关于接口中字段的一些约定：
**！！！字段注意不要随意扩充**

-   facility 程序/服务/标识应用名，如 loginsvr 、robot 等，初始化时使用WithLogSource函数进行配置
-   version 程序版本号 1.12.3
-   host 主机ip/pid 如：10.17.40.201/193923
-   tag 用于标识业务相关索引，如订单号、帐号等（优先使用订单号作主查询索引）
-   secondfacility 子类，标识日志类别，如登录耗时日志等

## 使用示例
**优先使用 zap模式**

### 1. zap模式
```go
package main

import (
        "git.17usoft.com/go/graylog"
        "go.uber.org/zap"
)

func main() {
        graylog.InitZapLog(graylog.ZapWithConnWriter("udp://10.17.43.201:12203", false),
                graylog.ZapWithFields([]zap.Field{zap.String("host", "10.17.40.201/193923"), zap.String("facility", "robot"), zap.String("_secondfacility", "qp")}),
                graylog.ZapWithLogPath("./logs"),
                graylog.ZapWithTCPMsgSplit(graylog.TCPModeLogstash),
                graylog.ZapWithAtomicLevel(graylog.DEBUG),
                graylog.ZapWithRotateType(graylog.SizeDivision),
                graylog.ZapWithAESEncrypt([]byte("justatest"), []byte{0x5a, 0xe3, 0xf0, 0x46, 0xcc, 0x11, 0xb4, 0x45, 0x09, 0x04, 0x47, 0x58, 0x00, 0xbf, 0x88, 0xd5}),
                graylog.ZapWithEncryptFields([]string{"paypassportseno", "mobile", "payrealname", `(?i)mobile_? ?no`, `(?i)user_? ?name`, "password", "id_no", "passportseno", "student_no"}, 1),
        )
        //graylog.Logger = graylog.Logger.WithOptions(zap.Fields(zap.String("robot_version", "1.72134"), zap.String("appVersion", "5.1.13")))
        graylog.Logger.Debug("interface DEBUG "+"请求12306返回 %s", "error.html")
}
```
### 2. 普通模式
***该模式准备废弃***
```go
package main

import (
	"fmt"
	"time"

	"git.17usoft.com/go/graylog"
)

func testlog() {
	if err :=graylog.NewGraylogClient(graylog.WithLogSource("109.20.13.30/83129/serverid"),
		graylog.WithLogType("loginServer"),
		graylog.WithQueueMaxLen(100),
		graylog.WithSendTaskNum(5),
		//	graylog.WithConnWriter("tcp", "10.17.40.201:12202", false),
		//	graylog.WithConnWriter("udp", "10.17.40.201:12203", false),
		graylog.WithHttpWriter("http://10.17.40.21:12201/gelf")); err!=nil{
		fmt.Println(err)
		return
	}
	graylog.InfoEx("CreateOrder", "TGT_test_77193134", 1290, "创建订单失败:%s",
		`{"resultCode":"env save info error","resultData":{"dynamicNum":"2","dynamicTrace":"UXQ4uqhbeP8Hl56Nwv5HAeIrKUue5MZjh1QbhNoEr+oBa9XxtLLye2hKlsZ5AQAA","resultType":"dynamic"},"success":false}`)

	time.Sleep(10 * time.Millisecond)
}
```
## TODO
- 剥离字段加密
- 优化TCP发送
- 完善TLS发送逻辑

