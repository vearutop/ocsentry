package ocsentry

import (
	"context"

	"github.com/getsentry/sentry-go"
	"go.opencensus.io/trace"
)

// WrapTracer creates a tracer that manages Sentry spans together with OpenCensus spans.
func WrapTracer(tracer trace.Tracer) trace.Tracer {
	if _, ok := tracer.(sentryTracer); ok {
		return tracer
	}

	return sentryTracer{Tracer: tracer}
}

type sentryTracer struct {
	trace.Tracer
}

type contextKey struct{}

// FromContext returns the Span stored in a context, or nil if there isn't one.
func (st sentryTracer) FromContext(ctx context.Context) *trace.Span {
	s, _ := ctx.Value(contextKey{}).(*trace.Span) // nolint:errcheck // Failed assertion ends up with nil.

	return s
}

// NewContext returns a new context with the given Span attached.
func (st sentryTracer) NewContext(parent context.Context, s *trace.Span) context.Context {
	return context.WithValue(parent, contextKey{}, s)
}

func (st sentryTracer) startSpan(ctx context.Context, name string, span *trace.Span, parent trace.SpanContext) (context.Context, *trace.Span) {
	var (
		sc = span.SpanContext()
		ss *sentry.Span
	)

	hasParent := parent != trace.SpanContext{}

	if !hasParent {
		ss = sentry.StartSpan(ctx, name, sentry.TransactionName(name))
	} else {
		ss = sentry.StartSpan(ctx, name)
	}

	ss.Status = sentry.SpanStatusOK
	ss.TraceID = sentry.TraceID(sc.TraceID)
	ss.SpanID = sentry.SpanID(sc.SpanID)

	if hasParent {
		ss.ParentSpanID = sentry.SpanID(parent.SpanID)
	}

	if sc.IsSampled() {
		ss.Sampled = sentry.SampledTrue
	}

	wrappedSpan := trace.NewSpan(sentrySpan{Span: span, ss: ss})

	return st.NewContext(ss.Context(), wrappedSpan), wrappedSpan
}

func (st sentryTracer) StartSpanWithRemoteParent(ctx context.Context, name string, parent trace.SpanContext, o ...trace.StartOption) (context.Context, *trace.Span) {
	ctx, span := st.Tracer.StartSpanWithRemoteParent(ctx, name, parent, o...)

	return st.startSpan(ctx, name, span, parent)
}

func (st sentryTracer) StartSpan(ctx context.Context, name string, o ...trace.StartOption) (context.Context, *trace.Span) {
	parent := st.Tracer.FromContext(ctx)

	ctx, span := st.Tracer.StartSpan(ctx, name, o...)
	if parent == nil {
		return st.startSpan(ctx, name, span, trace.SpanContext{})
	}

	return st.startSpan(ctx, name, span, parent.SpanContext())
}
