package krecovery

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	ecode "github.com/XuThreeFire/goutil/errorx"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
)

var ServerHandler HandlerFunc = func(ctx context.Context, req, err interface{}) error {
	return ecode.ErrInternalError.AddMsg(fmt.Sprintf("%+v", err))
}

// HandlerFunc is recovery handler func.
type HandlerFunc func(ctx context.Context, req, err interface{}) error

// Option is recovery option.
type Option func(*options)

type options struct {
	handler HandlerFunc
	logger  log.Logger
}

// WithHandler with recovery handler.
func WithHandler(h HandlerFunc) Option {
	return func(o *options) {
		o.handler = h
	}
}

// WithLogger with recovery logger.
func WithLogger(logger log.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// Recovery is a server middlewarex that recovers from any panics.
func Recovery(opts ...Option) middleware.Middleware {
	op := options{
		logger: log.DefaultLogger,
		handler: func(ctx context.Context, req, err interface{}) error {
			return errors.New(fmt.Sprintf("panic:%+v", err))
		},
	}
	for _, o := range opts {
		o(&op)
	}
	logger := log.NewHelper(op.logger)
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			defer func() {
				if rerr := recover(); rerr != nil {
					buf := make([]byte, 64<<10) //nolint:gomnd
					n := runtime.Stack(buf, false)
					buf = buf[:n]
					logger.WithContext(ctx).Errorf("%v: %+v\n%s\n", rerr, req, buf)

					err = op.handler(ctx, req, rerr)
				}
			}()
			return handler(ctx, req)
		}
	}
}
