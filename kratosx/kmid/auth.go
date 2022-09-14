package kmid

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"strings"

	ecode "github.com/XuThreeFire/goutil/errorx"
	klog "github.com/XuThreeFire/goutil/kratosx/klog"
	time_parse "github.com/XuThreeFire/goutil/timex"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/uuid"
)

type validator interface {
	Validate() error
}

type SignKey struct{}

type SignInfo struct {
	User      string
	Method    string
	Timestamp string
	Body      []byte
	Key       string
}

func (s *SignInfo) CalculateSign() string {
	signData := []byte(s.User + s.Method + s.Timestamp)
	signData = append(signData, s.Body...)
	signData = append(signData, []byte(s.Key)...)
	md5Bytes := md5.Sum(signData)
	return strings.ToLower(hex.EncodeToString(md5Bytes[:]))
}

// AuthHttp is the function type used for http custom validators.
func AuthHttp(userMap map[string]struct{}, key, testSign string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			hc, ok := ctx.(http.Context)
			if !ok {
				return nil, ecode.ErrInternalError
			}

			r := hc.Request()
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, ecode.ErrEmptyParam
			}
			// defer r.Body.Close()

			// set trace_id
			traceID := r.Header.Get("Trace-Id")
			ctx = context.WithValue(ctx, klog.TraceIDKey{}, traceID)
			hc.Response().Header().Set("Trace-Id", traceID)
			authUser := r.Header.Get("Auth-User")
			if _, isOk := userMap[authUser]; !isOk {
				return nil, ecode.ErrIllegaUser
			}

			method := r.Header.Get("Method")
			timestamp := r.Header.Get("Timestamp")
			signature := r.Header.Get("Signature")

			s := SignInfo{
				User:      authUser,
				Method:    method,
				Timestamp: timestamp,
				Body:      bodyBytes,
				Key:       key,
			}
			sign := s.CalculateSign()
			if signature != sign {
				if signature != testSign {
					return nil, ecode.ErrSignatureError
				}
			}

			if v, ok := req.(validator); ok {
				if err := v.Validate(); err != nil {
					return nil, ecode.ErrIllegalData
				}
			}

			return handler(ctx, req)
		}
	}
}

// AuthGrpc is the function type used for grpc custom validators.
func AuthGrpc(userMap map[string]struct{}, key, testSign string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return ecode.ErrInternalError, nil
			}

			// set trace_id
			traceID := tr.RequestHeader().Get("Trace-Id")
			ctx = context.WithValue(ctx, klog.TraceIDKey{}, traceID)
			tr.ReplyHeader().Set("Trace-Id", traceID)

			authUser := tr.RequestHeader().Get("Auth-User")
			if _, isOk := userMap[authUser]; !isOk {
				return nil, ecode.ErrIllegaUser
			}
			method := tr.RequestHeader().Get("Method")
			timestamp := tr.RequestHeader().Get("Timestamp")
			signature := tr.RequestHeader().Get("Signature")

			s := SignInfo{
				User:      authUser,
				Method:    method,
				Timestamp: timestamp,
				Body:      nil,
				Key:       key,
			}
			sign := s.CalculateSign()
			if signature != sign {
				if signature != testSign {
					return ecode.ErrSignatureError, nil
				}
			}

			if v, ok := req.(validator); ok {
				if err := v.Validate(); err != nil {
					return ecode.ErrIllegalRequest, nil
				}
			}

			return handler(ctx, req)
		}
	}
}

func AuthHttpClient(user, key string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			s, ok := ctx.Value(SignKey{}).(*SignInfo)
			if !ok {
				return nil, errors.New("sign info not found")
			}
			if tr, ok := transport.FromClientContext(ctx); ok {
				header := tr.RequestHeader()

				traceID := func() string {
					if traceID, ok := ctx.Value(klog.TraceIDKey{}).(string); ok {
						return traceID
					}
					return uuid.NewString()
				}()

				method := func() string {
					words := strings.Split(tr.Operation(), "/")
					if len(words) > 0 {
						return words[len(words)-1]
					}
					return "errorx method"
				}()
				tm := time_parse.GetTimeStamp()

				if s.User == "" {
					s.User = user
				}
				if s.Key == "" {
					s.Key = key
				}
				s.Method = method
				s.Timestamp = tm
				sign := s.CalculateSign()

				header.Set("Auth-User", user)
				header.Set("Method", method)
				header.Set("Timestamp", tm)
				header.Set("Signature", sign)
				header.Set("Trace-Id", traceID)
			}
			return handler(ctx, req)
		}
	}
}
