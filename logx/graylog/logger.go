package graylog

import (
	"crypto/tls"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	Minute = "minute"
	Hour   = "hour"
	Day    = "day"
	Month  = "month"
	Year   = "year"
)

const (
	INFO  = "INFO"
	DEBUG = "DEBUG"
	WARN  = "WARN"
	ERROR = "ERROR"
)

const (
	TCPModeLogstash = "logstash"
	TCPModeGraylog  = "graylog"
)

var (
	Logger           *zap.Logger // zap logger
	logOpts          *logOptions // for encryptFields
	customizedWriter []writerInterface
	logOptMtx        sync.Mutex
	zapmx            sync.Mutex
	loggerSugar      *zap.SugaredLogger
)

type RotateType string

const (
	TimeDivision RotateType = "time"
	SizeDivision RotateType = "size"
)

var _encoderNameToConstructor = map[string]func(zapcore.EncoderConfig) zapcore.Encoder{
	"console": func(encoderConfig zapcore.EncoderConfig) zapcore.Encoder {
		return zapcore.NewConsoleEncoder(encoderConfig)
	},
	"json": func(encoderConfig zapcore.EncoderConfig) zapcore.Encoder {
		return zapcore.NewJSONEncoder(encoderConfig)
	},
}

type timeUnit string

type logOptions struct {
	Encoding      string // 输出格式 "json" 或者 "console"
	Dir           string
	InfoFilename  string // info级别日志文件名
	ErrorFilename string // warn级别日志文件名

	Division RotateType // 归档方式

	TimeUnit timeUnit // 时间归档 切割单位

	MaxSize    int  // 每个日志文件保存的最大尺寸 单位：M
	MaxBackups int  // 日志文件最多保存多少个备份
	MaxAge     int  // 文件最多保存多少天
	Compress   bool // 是否压缩

	LevelSeparate bool            // 是否日志分级
	stdoutDisplay bool            // 是否在控制台输出
	caller        bool            // 是否输出文件行号
	stack         bool            // 是否输出堆栈信息
	callSkip      int             // 调用深度
	atomicLevel   zap.AtomicLevel // 网络日志级别
	localLogLevel zap.AtomicLevel // 本地日志文件级别
	FixFields     []zap.Field

	isGELF bool // 是否为graylog的GELF
	GELF   struct {
		net            string
		addr           string
		reconnectOnMsg bool
		config         *tls.Config
		tcpMsgMode     string // graylog 和 logstash tcp分割为 \x00 和 \n
		sendBufChNum   int    //write data async goroutine num
	}

	encryptFields []string        // 需要加密的字段
	encryptDepth  int             // 加密的层级，默认为1。0 代表无转义,1 层代表一次转义
	cryptor       cryptoInterface // crypto(md5/aes)
}

// InitZapLog InitLog 日志初始化
// 注意！此调用会覆盖原Logger指针的对象
func InitZapLog(opts ...ZapClientOptions) error {
	var c *logOptions = newLog()

	// 设置初始默认级别
	c.atomicLevel.UnmarshalText([]byte("debug"))
	c.localLogLevel.UnmarshalText([]byte("debug"))

	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	if c.Division == TimeDivision {
		if c.TimeUnit == "" {
			c.TimeUnit = Day
		}
		if c.MaxAge <= 0 {
			c.MaxAge = 7
		}
	} else if c.Division == SizeDivision {
		c.MaxAge = 0

		if c.MaxSize <= 0 {
			c.MaxSize = 50
		}
		if c.MaxBackups <= 0 {
			c.MaxBackups = 7
		}
	}

	c.InfoFilename = c.Dir + "/info.log"
	c.ErrorFilename = c.Dir + "/err.log"

	// create logger handler
	err := c.initLogger()
	logOptMtx.Lock()
	logOpts = c
	logOptMtx.Unlock()
	return err
}

//Sync 同步日志
func Sync() {
	if Logger == nil {
		return
	}

	Logger.Sync()

	if loggerSugar != nil {
		loggerSugar.Sync()
	}
	for _, w := range customizedWriter {
		if w != nil {
			w.Stop()
		}
	}
	customizedWriter = customizedWriter[:0]
}

func newLog() *logOptions {
	return &logOptions{
		Encoding:      "console",
		InfoFilename:  "./logs/info.log",
		ErrorFilename: "./logs/err.log",
		Division:      SizeDivision,
		TimeUnit:      Day,
		MaxSize:       100,
		MaxBackups:    10,
		MaxAge:        15,
		Compress:      true,
		LevelSeparate: false,
		stdoutDisplay: false,
		caller:        true,
		stack:         false,
		isGELF:        false,
	}
}

func (c *logOptions) initLogger() error {
	if c == nil {
		return errors.New("logOptions is nil")
	}

	var (
		core                     zapcore.Core
		infoHook, warnHook       io.Writer
		wsInfo, wsWarn, wsNetLog []zapcore.WriteSyncer
	)

	if c.Encoding == "" {
		c.Encoding = "console"
	}
	encoder := _encoderNameToConstructor[c.Encoding]

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "logtime",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "linenum", //"caller"
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder, // 大写编码器INFO, 小写zapcore.LowercaseLevelEncoder
		EncodeTime:     timeEncoder,                 // zapcore.ISO8601TimeEncoder,  ISO8601 UTC 时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder, // zapcore.FullCallerEncoder,  全路径编码器
		EncodeName:     zapcore.FullNameEncoder,
	}

	if c.stdoutDisplay {
		wsInfo = append(wsInfo, zapcore.AddSync(os.Stdout))
		// wsWarn = append(wsWarn, zapcore.AddSync(os.Stdout))
	}

	// zapcore WriteSyncer setting
	if c.ErrorFilename != "" {
		switch c.Division {
		case TimeDivision:
			err := os.MkdirAll(filepath.Dir(c.ErrorFilename), 0744)
			if err != nil {
				// panic("can't make directories for new logfile")
				return err
			}
			infoHook, err = c.timeDivisionWriter(c.ErrorFilename)
			if err != nil {
				return err
			}
			if c.LevelSeparate {
				err := os.MkdirAll(filepath.Dir(c.ErrorFilename), 0744)
				if err != nil {
					// panic("can't make directories for new logfile")
					return err
				}
				warnHook, err = c.timeDivisionWriter(c.ErrorFilename)
				if err != nil {
					return err
				}
			}
		case SizeDivision:
			var err error
			if infoHook, err = c.sizeDivisionWriter(c.ErrorFilename); err != nil {
				return err
			}
			if c.LevelSeparate {
				if warnHook, err = c.sizeDivisionWriter(c.ErrorFilename); err != nil {
					return err
				}
			}
		}

		if infoHook != nil {
			wsInfo = append(wsInfo, zapcore.AddSync(infoHook))
		}
	}

	if c.ErrorFilename != "" {
		if warnHook != nil {
			wsWarn = append(wsWarn, zapcore.AddSync(warnHook))
		}
	}

	// Separate info and warning log
	zapCores := []zapcore.Core{
		zapcore.NewCore(encoder(encoderConfig), zapcore.NewMultiWriteSyncer(wsInfo...), c.localLogLevel),
	}
	if c.LevelSeparate {
		zapCores = append(zapCores, zapcore.NewCore(encoder(encoderConfig), zapcore.NewMultiWriteSyncer(wsWarn...), warnLevel()))
	}
	tmpWInterface := []writerInterface{}
	if c.isGELF {
		jsonEncoder := _encoderNameToConstructor["json"]
		netWriter := newConnWriter(c.GELF.net, c.GELF.addr, c.GELF.reconnectOnMsg)
		if len(c.encryptFields) > 0 && c.cryptor != nil {
			if err := netWriter.setEncOpt(c.encryptFields, c.encryptDepth, c.cryptor); err != nil {
				return err
			}
		}
		if c.GELF.net == "tcp" && c.GELF.tcpMsgMode != "" {
			netWriter.setTCPMsgMode(c.GELF.tcpMsgMode)
		}

		tmpWInterface = append(tmpWInterface, netWriter)

		wsNetLog = append(wsNetLog, netWriter)
		core := zapcore.NewCore(jsonEncoder(encoderConfig), zapcore.NewMultiWriteSyncer(wsNetLog...), c.atomicLevel)
		if len(c.FixFields) > 0 {
			core = core.With(c.FixFields)
		}
		// zapCores = append(zapCores, zapcore.NewCore(json_encoder(encoderConfig), zapcore.NewMultiWriteSyncer(wsNetlog...), zap.DebugLevel))
		zapCores = append(zapCores, core)
	}
	core = zapcore.NewTee(zapCores...)

	// file line number display
	development := zap.Development()
	stackTrace := zap.AddStacktrace(zapcore.WarnLevel)

	// init default key
	opts := []zap.Option{development}
	if c.caller {
		opts = append(opts, zap.AddCaller())
	}
	if c.stack {
		opts = append(opts, stackTrace)
	}
	if c.callSkip != 0 {
		opts = append(opts, zap.AddCallerSkip(c.callSkip))
	}

	zapmx.Lock()
	//	if Logger != nil {
	//		Logger.Sync()
	//	}
	Sync()
	Logger = zap.New(core, opts...)
	customizedWriter = append(customizedWriter, tmpWInterface...)
	zapmx.Unlock()

	if Logger == nil {
		return errors.New("error initializing logger")
	}
	loggerSugar = Logger.WithOptions(zap.AddCallerSkip(1)).Sugar()

	return nil

	// if logger == nil {
	// 	logger = zap.New(core, opts...)
	// } else {
	// 	// 需要验证下会不会更新 core
	// 	opts = append(opts,
	// 		zap.WrapCore(func(core zapcore.Core) zapcore.Core { return core }))
	// 	logger.Sync() // sync cache
	// 	// clone old logger and apply options
	// 	logger = logger.WithOptions(opts...)
	// }
}

func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

func infoLevel() zap.LevelEnablerFunc {
	return func(lvl zapcore.Level) bool {
		return lvl < zapcore.WarnLevel
	}
}

func warnLevel() zap.LevelEnablerFunc {
	return func(lvl zapcore.Level) bool {
		return lvl >= zapcore.WarnLevel
	}
}

func (c *logOptions) sizeDivisionWriter(filename string) (io.Writer, error) {
	if c == nil {
		return nil, errors.New("log options not set")
	}

	hook := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
		Compress:   c.Compress,
		LocalTime:  true,
	}
	return hook, nil
}

func (c *logOptions) timeDivisionWriter(filename string) (io.Writer, error) {
	if c == nil {
		return nil, errors.New("log options not set")
	}

	hook, err := rotatelogs.New(
		filename+c.TimeUnit.format(),
		rotatelogs.WithMaxAge(time.Duration(int64(24*time.Hour)*int64(c.MaxAge))),
		rotatelogs.WithRotationTime(c.TimeUnit.rotationGap()),
	)
	if err != nil {
		// panic(err)
		return nil, err
	}
	return hook, nil
}

func (t timeUnit) format() string {
	switch t {
	case Minute:
		return ".%Y%m%d%H%M"
	case Hour:
		return ".%Y%m%d%H"
	case Day:
		return ".%Y%m%d"
	case Month:
		return ".%Y%m"
	case Year:
		return ".%Y"
	default:
		return ".%Y%m%d"
	}
}

func (t timeUnit) rotationGap() time.Duration {
	switch t {
	case Minute:
		return time.Minute
	case Hour:
		return time.Hour
	case Day:
		return time.Hour * 24
	case Month:
		return time.Hour * 24 * 30
	case Year:
		return time.Hour * 24 * 365
	default:
		return time.Hour * 24
	}
}
