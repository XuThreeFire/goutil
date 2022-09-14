package klog

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

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
