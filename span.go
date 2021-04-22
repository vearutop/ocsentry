package ocsentry

import (
	"github.com/getsentry/sentry-go"
	"go.opencensus.io/trace"
)

type sentrySpan struct {
	*trace.Span
	ss *sentry.Span
}

func (ss sentrySpan) End() {
	ss.ss.Finish()
	ss.Span.End()
}

func (ss sentrySpan) AddAttributes(attributes ...trace.Attribute) {
	if ss.ss.Data == nil {
		ss.ss.Data = make(map[string]interface{}, len(attributes))
	}

	for _, attr := range attributes {
		ss.ss.Data[attr.Key()] = attr.Value()
	}

	ss.Span.AddAttributes(attributes...)
}

func (ss sentrySpan) SetStatus(status trace.Status) {
	ss.ss.Status = sentry.SpanStatus(status.Code + 1)
	ss.Span.SetStatus(status)
}
