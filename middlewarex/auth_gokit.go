package midutil

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	errorutil "github.com/XuThreeFire/goutil/errorx"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"time"
)

//var (
//	// ErrSignature 102 签名验证错误
//	ErrSignature = errorutil.New(102, "签名验证错误", false)
//	// ErrInvalidAuthUser 103 未授权用户
//	ErrInvalidAuthUser = errors.New("未授权用户", errors.WithCode(103))
//	// ErrTimestamp 106 签名过期
//	ErrTimestamp = errors.New("签名过期", errors.WithCode(106))
//)

// Signature 签名算法
// md5sum(AuthUser+Method+Timestamp+{request body}+signKey)) 小写

// AuthError represents an authorization error.
type AuthError struct {
	code    int
	message string
}

// BusinessCode 业务码
func (a AuthError) BusinessCode() int {
	return a.code
}

// Error is an implementation of the Error interface.
func (a AuthError) Error() string {
	return a.message
}

// AuthMiddleware returns Authentication middleware for a private sign
func AuthMiddleware(authUserMap map[string]bool, signKey string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {

			// authUser
			reqAuthUser, ok := ctx.Value(ContextKeyRequestAuthUser).(string)
			if !ok {
				return nil, errorutil.ErrIllegaUser
			}
			if authUserMap[reqAuthUser] == false {
				return nil, errorutil.ErrIllegaUser
			}
			// signature
			signature, ok := ctx.Value(ContextKeyRequestSignature).(string)
			if !ok {
				return nil, errorutil.ErrSignatureError
			}
			calculateSignature, ok := ctx.Value(ContextKeyCalculateSignature).(string)
			if !ok {
				return nil, errorutil.ErrSignatureError
			}
			if signature != calculateSignature {
				return nil, errorutil.ErrSignatureError
			}

			// timestamp
			timestampStr, ok := ctx.Value(ContextKeyRequestTimestamp).(string)
			if !ok {
				return nil, errorutil.ErrExpiredSignature
			}
			timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				return nil, errorutil.ErrExpiredSignature
			}
			diff := time.Now().UnixMilli() - timestamp

			// TODO: 时间差配置
			if math.Abs(float64(diff)) > 120*1000 {
				return nil, errors.New(fmt.Sprintf("diff=%v, err=%s", diff, errorutil.ErrExpiredSignature.Error()))
			}
			return next(ctx, request)
		}
	}
}

// CalculateSignatureToContext returns an kithttp.HandlerFunc that context wraps the sign parameters.
func CalculateSignatureToContext(myUser, signKey string) kithttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		var bodyBytes []byte
		if r.Method != "GET" {
			bodyBytes, _ = ioutil.ReadAll(r.Body)
			r.Body.Close()
			r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		authUser := r.Header.Get(string(ContextKeyRequestAuthUser))
		method := r.Header.Get(string(ContextKeyRequestMethod))
		timestamp := r.Header.Get(string(ContextKeyRequestTimestamp))
		signature := r.Header.Get(string(ContextKeyRequestSignature))
		//calculateSignature := CalculateSignature(myUser, method, timestamp, bodyBytes, signKey)
		calculateSignature := CalculateSignature(authUser, method, timestamp, bodyBytes, signKey)

		ctx = context.WithValue(ctx, ContextKeyRequestTimestamp, timestamp)
		ctx = context.WithValue(ctx, ContextKeyRequestMethod, method)
		ctx = context.WithValue(ctx, ContextKeyRequestAuthUser, authUser)
		ctx = context.WithValue(ctx, ContextKeyRequestSignature, signature)
		ctx = context.WithValue(ctx, ContextKeyCalculateSignature, calculateSignature)

		return ctx
	}
}

// CalculateSignature 计算签名
// md5sum(AuthUser+Method+Timestamp+{request body}+signKey)) 小写
func CalculateSignature(authUser, method, timestamp string, bodyBytes []byte, signKey string) string {
	h := md5.New()
	h.Write([]byte(authUser + method + timestamp + string(bodyBytes) + signKey))
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateSignatureToRequest 生成签名headers字段
func GenerateSignatureToRequest(authUser, signKey, method string) kithttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		var bodyBytes []byte
		if r.Method != "GET" {
			bodyBytes, _ = ioutil.ReadAll(r.Body)
			r.Body.Close()
			r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		} else {
			bodyBytes = nil
		}

		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		calculateSignature := CalculateSignature(authUser, method, timestamp, bodyBytes, signKey)
		r.Header.Set(string(ContextKeyRequestAuthUser), authUser)
		r.Header.Set(string(ContextKeyRequestMethod), method)
		r.Header.Set(string(ContextKeyRequestTimestamp), timestamp)
		r.Header.Set(string(ContextKeyRequestSignature), calculateSignature)
		return ctx
	}
}

func MethodFromContext(ctx context.Context) string {
	method, ok := ctx.Value(ContextKeyRequestMethod).(string)
	if ok {
		return method
	}
	return ""
}

func AuthUserFromContext(ctx context.Context) string {
	val := ctx.Value(ContextKeyRequestAuthUser)
	if authUser, ok := val.(string); ok {
		return authUser
	}
	return "default"
}
func PartnerIdFromContext(ctx context.Context) string {
	val := ctx.Value(ContextKeyPartnerId)
	if partnerId, ok := val.(string); ok {
		return partnerId
	}
	return "default"
}

func UserNameFromContext(ctx context.Context) string {
	val := ctx.Value(ContextKeyUserName)
	if userName, ok := val.(string); ok {
		return userName
	}
	return ""
}
func UserKeyFromContext(ctx context.Context) string {
	val := ctx.Value(ContextKeyUserKey)
	if userKey, ok := val.(string); ok {
		return userKey
	}
	return ""
}
func TraceIdFromContext(ctx context.Context) string {
	val := ctx.Value(ContextKeyTraceId)
	if traceId, ok := val.(string); ok {
		return traceId
	}
	return ""
}

func InterfaceKeyFromContext(ctx context.Context) string {
	val := ctx.Value(ContextKeyInterfaceKey)
	if interfaceKey, ok := val.(string); ok {
		return interfaceKey
	}
	return "default"
}

// GenerateSignatureToRequestForTcServer 生成签名headers字段
func GenerateSignatureToRequestForTcServer(authPartnerId, authInterfaceKey, method string) kithttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		var bodyBytes []byte
		if r.Method != "GET" {
			bodyBytes, _ = ioutil.ReadAll(r.Body)
			r.Body.Close()
			r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		} else {
			bodyBytes = nil
		}

		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		calculateSignature := CalculateSignatureForTcServer(authPartnerId, authInterfaceKey, method, timestamp)
		r.Header.Set(string(ContextKeyRequestAuthUser), authPartnerId)
		r.Header.Set(string(ContextKeyRequestMethod), method)
		r.Header.Set(string(ContextKeyRequestTimestamp), timestamp)
		r.Header.Set(string(ContextKeyRequestSignature), calculateSignature)

		return ctx
	}
}

// CalculateSignature 计算签名
// md5sum(AuthUser+Method+Timestamp+{request body}+signKey)) 小写
func CalculateSignatureForTcServer(authPartnerId, authInterfaceKey, method, reqTimestamp string) string {
	h := md5.New()
	h.Write([]byte(fmt.Sprintf("%s%s%s", authPartnerId, method, reqTimestamp, md5.Sum([]byte(authInterfaceKey)))))
	return hex.EncodeToString(h.Sum(nil))
}

// ContextWithPartnerId context wraps the request id
func ContextWithPartnerId(ctx context.Context, partnerId string) context.Context {
	return context.WithValue(ctx, ContextKeyPartnerId, partnerId)
}

// ContextWithInterfaceKey context wraps the request id
func ContextWithInterfaceKey(ctx context.Context, interfaceKey string) context.Context {
	return context.WithValue(ctx, ContextKeyInterfaceKey, interfaceKey)
}

// ContextWithMethod context wraps the request id
func ContextWithMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestMethod, method)
}

func ContextWithUserKey(ctx context.Context, userKey string) context.Context {
	return context.WithValue(ctx, ContextKeyUserKey, userKey)
}

func ContextWithUserName(ctx context.Context, userName string) context.Context {
	return context.WithValue(ctx, ContextKeyUserName, userName)
}
func ContextWithTraceId(ctx context.Context, traceId string) context.Context {
	return context.WithValue(ctx, ContextKeyTraceId, traceId)
}
