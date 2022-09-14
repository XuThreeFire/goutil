package midutil

import (
	"context"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

// GenerateTraceID generate a trace id
func GenerateTraceID() string {
	traceID := uuid.New().String()
	return strings.ReplaceAll(traceID, "-", "")
}

// TraceIDFormContext get trace id form context
func TraceIDFormContext(ctx context.Context) string {
	val := ctx.Value(ContextKeyRequestTraceID)
	if traceID, ok := val.(string); ok {
		return traceID
	}
	return GenerateTraceID()
}
func RequestIDFormContext(ctx context.Context) string {
	val := ctx.Value(ContextKeyRequestTraceID)
	if requestID, ok := val.(string); ok {
		return requestID
	}
	return ""
}

// ContextWithTraceID context wraps the trace id
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestTraceID, traceID)
}

// ContextWithRequestID context wraps the request id
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestXRequestID, requestID)
}

// HTTPToContext returns an http.HandlerFunc that context wraps the traceId
// 从请求里面提取traceId、requestId
func HTTPToContext() kithttp.RequestFunc {
	return func(ctx context.Context, req *http.Request) context.Context {
		// trace-id
		traceID := req.Header.Get(string(ContextKeyRequestTraceID))
		if traceID == "" {
			traceID = GenerateTraceID()
		}
		ctx = context.WithValue(ctx, ContextKeyRequestTraceID, traceID)

		// x-request-id
		requestID := req.Header.Get(string(ContextKeyRequestXRequestID))
		if requestID == "" {
			requestID = uuid.New().String()
		}
		ctx = context.WithValue(ctx, ContextKeyRequestXRequestID, requestID)
		return ctx
	}
}

// ContextToHTTPRequest returns an http RequestFunc that injects a traceId
// 给发出去的Request添加TraceId和requestId
func ContextToHTTPRequest() kithttp.RequestFunc {
	return func(ctx context.Context, req *http.Request) context.Context {
		// Trace-Id
		val := ctx.Value(ContextKeyRequestTraceID)
		if traceID, ok := val.(string); ok {
			req.Header.Set(string(ContextKeyRequestTraceID), traceID)
		} else {
			newTraceID := GenerateTraceID()
			req.Header.Set(string(ContextKeyRequestTraceID), newTraceID)
		}
		// X-Request-Id, retry 时复用同一个 requestId
		val = ctx.Value(ContextKeyRequestXRequestID)
		if requestID, ok := val.(string); ok {
			req.Header.Set(string(ContextKeyRequestXRequestID), requestID)
		} else {
			newRequestID := uuid.New().String()
			req.Header.Set(string(ContextKeyRequestXRequestID), newRequestID)
		}
		//

		return ctx
	}
}

// ContextToHTTPResponse  returns an http.HandlerFunc that context wraps the traceId
// 给接收的请求，返回traceId
func ContextToHTTPResponse() kithttp.ServerResponseFunc {
	return func(ctx context.Context, w http.ResponseWriter) context.Context {
		// Trace-Id
		val := ctx.Value(ContextKeyRequestTraceID)
		if traceID, ok := val.(string); ok {
			w.Header().Set(string(ContextKeyRequestTraceID), traceID)
		}
		// X-Request-Id
		val = ctx.Value(ContextKeyRequestXRequestID)
		if requestID, ok := val.(string); ok {
			w.Header().Set(string(ContextKeyRequestXRequestID), requestID)
		}
		return ctx
	}
}

type TraceIDKey struct{}

// TraceID returns a trace_id valuer.
func TraceID() log.Valuer {
	return func(ctx context.Context) interface{} {
		if ctx != nil {
			if info, ok := transport.FromServerContext(ctx); ok {
				if traceID := info.ReplyHeader().Get("Trace-Id"); traceID != "" {
					return traceID
				}
			}
			if traceID, ok := ctx.Value(TraceIDKey{}).(string); ok {
				return traceID
			}
		}
		return ""
		// return "unknown-" + uuid.New().String()
	}
}
