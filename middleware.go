package ocsentry

import (
	"context"
	"errors"
	"net/http"

	"github.com/getsentry/sentry-go"
)

// HTTPHandlerMiddleware prepares request context without creating sentry trace span
// for trace span is to be created by OpenCensus instrumentation.
func HTTPHandlerMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		hub := sentry.CurrentHub().Clone()
		ctx = sentry.SetHubOnContext(ctx, hub)

		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OnPanic handles recovered panics.
func OnPanic(ctx context.Context, rcv interface{}) {
	hub := sentry.GetHubFromContext(ctx)

	if hub == nil {
		hub = sentry.CurrentHub()
	}

	hub.RecoverWithContext(ctx, rcv)
}

// ErrorHandler is a handler for ErrorCatcher of github.com/swaggest/usecase.
func ErrorHandler(ctx context.Context, input interface{}, err error) {
	hub := sentry.GetHubFromContext(ctx)

	if hub == nil {
		hub = sentry.CurrentHub()
	}

	hub.WithScope(func(scope *sentry.Scope) {
		var se interface {
			Fields() map[string]interface{}
		}
		if errors.As(err, &se) {
			scope.SetExtras(se.Fields())
		}

		scope.SetExtra("input", input)

		hub.CaptureException(err)
	})
}
