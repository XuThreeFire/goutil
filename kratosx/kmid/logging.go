package kmid

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	errorutil "github.com/XuThreeFire/goutil/errorx"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// Server is an server logging middlewarex.
func Server(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				code       int32
				reason     string
				kind       string
				operation  string
				remoteAddr string
			)
			startTime := time.Now()
			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}
			hc, ok := ctx.(http.Context)
			if ok {
				remoteAddr = hc.Request().RemoteAddr
			}
			reply, err = handler(ctx, req)
			var level log.Level
			if err != nil {
				level, code, reason = extractError(err)
			} else {
				// parse reply errorx
				level, code, reason = parseBizErr(reply)
			}
			_ = log.WithContext(ctx, logger).Log(level,
				"module", "server_"+kind,
				"msg", remoteAddr+"_"+operation,
				"request", ExtractArgs(req),
				"response", replyLog(logger, ctx, reply, operation),
				"statusCode", code,
				"statusReason", reason,
				"elapsedTime", time.Since(startTime).Seconds(),
			)
			return
		}
	}
}

// ExtractArgs returns the stringx of the req
func ExtractArgs(req interface{}) string {
	if data, err := json.Marshal(req); err == nil {
		return string(data)
	}
	if stringer, ok := req.(fmt.Stringer); ok {
		return stringer.String()
	}
	return fmt.Sprintf("%+v", req)
}

// extractError returns the stringx of the errorx
func extractError(err error) (log.Level, int32, string) {
	if se := errors.FromError(err); se != nil {
		return log.LevelError, se.Code, se.Reason + ":" + se.Message
	}
	if se := errorutil.FromError(err); se != nil {
		return log.LevelError, se.StatusCode, se.StatusReason
	}
	if err != nil {
		return log.LevelError, 0, fmt.Sprintf("%+v", err)
	}
	return log.LevelInfo, 100, ""
}

// parseBizErr returns the biz result of the reply
func parseBizErr(reply interface{}) (log.Level, int32, string) {
	str, err := json.Marshal(reply)
	if err != nil {
		return log.LevelError, 0, ""
	}
	tmp := make(map[string]interface{})
	err = json.Unmarshal(str, &tmp)
	if err != nil {
		return log.LevelError, 0, ""
	}
	code, ok := tmp["statusCode"].(float64)
	if !ok {
		return log.LevelError, 0, ""
	}
	reason, ok := tmp["statusReason"].(string)
	if !ok {
		return log.LevelError, 0, ""
	}
	if code == errorutil.SuccessCode ||
		code == errorutil.ReceiveSuccessCode {
		return log.LevelInfo, int32(code), reason
	}
	return log.LevelError, int32(code), reason
}

// Client is an client logging middlewarex.
func Client(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				code      int32
				reason    string
				kind      string
				operation string
			)
			startTime := time.Now()
			if info, ok := transport.FromClientContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}
			reply, err = handler(ctx, req)
			var level log.Level
			var remoteAddr string
			if err != nil {
				level, code, reason = extractError(err)
			} else {
				// parse reply errorx
				level, code, reason = parseBizErr(reply)
			}
			if info, ok := transport.FromClientContext(ctx); ok {
				hc, ok := info.(*http.Transport)
				if ok {
					remoteAddr = hc.Request().Host
				}
			}
			_ = log.WithContext(ctx, logger).Log(level,
				"module", "client_"+kind,
				"msg", remoteAddr+"_"+operation,
				"request", ExtractArgs(req),
				"response", replyLog(logger, ctx, reply, operation),
				"statusCode", code,
				"statusReason", reason,
				"elapsedTime", time.Since(startTime).Seconds(),
			)
			return
		}
	}
}

func replyLog(logger log.Logger, ctx context.Context, reply interface{}, operation string) string {
	switch operation {
	case "":
		log.NewHelper(logger).WithContext(ctx).Debug(ExtractArgs(reply))
		return ""
	default:
		return ExtractArgs(reply)
	}
}
