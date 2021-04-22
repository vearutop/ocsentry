package ocsentry_test

import (
	"log"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/vearutop/ocsentry"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
)

func ExampleWrapTracer() {
	// Initialize Sentry.
	err := sentry.Init(sentry.ClientOptions{
		Dsn:        "https://abc123abc123abc123abc123@o123456.ingest.sentry.io/1234567",
		ServerName: "my-service",
		Release:    "v1.2.3",
	})
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		sentry.Flush(time.Second)
	}()

	// Setup OC sampling.
	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.ProbabilitySampler(0.01),
	})

	// Enable Sentry wrapper.
	trace.DefaultTracer = ocsentry.WrapTracer(trace.DefaultTracer)

	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Hello, world!"))
		if err != nil {
			log.Print(err)

			return
		}
	})

	// Apply OpenCensus middleware and Sentry middlewares to your http.Handler.
	h = ocsentry.HTTPHandlerMiddleware(h)
	h = &ochttp.Handler{Handler: h}

	if err := http.ListenAndServe(":80", h); err != nil {
		log.Print(err)

		return
	}
}
