package ocsentry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	sen "github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/vearutop/ocsentry"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
)

func setupSentryMock(t *testing.T, requests []string) (deferFn func()) {
	t.Helper()

	i := 0
	sentryMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, []string{"/api/123123/envelope/", "/api/123123/store/"}, r.URL.String())
		assert.Equal(t, http.MethodPost, r.Method)

		b, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		lines := bytes.Split(b, []byte("\n"))
		if len(lines) == 4 {
			if len(requests) <= i {
				assertjson.EqualMarshal(t, []byte(`{"err": "unexpected request `+strconv.Itoa(i)+`"}`), json.RawMessage(lines[2]))

				return
			}

			assertjson.Equal(t, []byte(`{"event_id":"<ignore-diff>","sent_at":"<ignore-diff>"}`), lines[0])
			assertjson.Equal(t, []byte(`{"type":"transaction","length":"<ignore-diff>"}`), lines[1])
			assertjson.EqualMarshal(t, []byte(requests[i]), json.RawMessage(lines[2]))
		} else {
			if len(requests) <= i {
				assertjson.EqualMarshal(t, []byte(`{"err": "unexpected request `+strconv.Itoa(i)+`"}`), json.RawMessage(lines[0]))

				return
			}

			assertjson.EqualMarshal(t, []byte(requests[i]), json.RawMessage(lines[0]))
		}

		i++
	}))

	u, err := url.Parse(sentryMock.URL)
	require.NoError(t, err)

	u.User = url.UserPassword("foo", "")
	u.Path = "123123"

	assert.NoError(t, sen.Init(sen.ClientOptions{
		Dsn: u.String(),
	}))

	// Enable Sentry wrapper.
	d := trace.DefaultTracer
	trace.DefaultTracer = ocsentry.WrapTracer(trace.DefaultTracer, ocsentry.SkipTransactionNames("GET /health", "GET /metrics"))

	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.AlwaysSample(),
	})

	return func() {
		sen.Flush(time.Second)
		sentryMock.Close()
		assert.Equal(t, len(requests), i, "not all requests are received")

		trace.DefaultTracer = d
	}
}

func TestWrapTracer(t *testing.T) {
	defer setupSentryMock(t, []string{
		`{
  "contexts": {
    "device": "<ignore-diff>",
    "os": "<ignore-diff>",
    "runtime": "<ignore-diff>",
    "trace": {
      "trace_id": "<ignore-diff>",
      "span_id": "<ignore-diff>",
      "op": "s1",
      "status": "ok"
    }
  },
  "event_id": "<ignore-diff>",
  "level": "info",
  "platform": "go",
  "sdk": "<ignore-diff>",
  "server_name": "<ignore-diff>",
  "transaction": "s1",
  "user": {},
  "type": "transaction",
  "spans": [
    {
      "trace_id": "<ignore-diff>",
      "span_id": "<ignore-diff>",
      "op": "s2",
      "status": "ok",
      "start_timestamp": "<ignore-diff>",
      "timestamp": "<ignore-diff>",
      "parent_span_id": "<ignore-diff>"
    }
  ],
  "start_timestamp": "<ignore-diff>",
  "timestamp": "<ignore-diff>"
}`,
	})()

	ctx, s1 := trace.StartSpan(context.Background(), "s1")

	time.Sleep(10 * time.Millisecond)

	_, s2 := trace.StartSpan(ctx, "s2")

	time.Sleep(10 * time.Millisecond)

	s2.End()

	time.Sleep(10 * time.Millisecond)

	s1.End()

	_, s3 := trace.StartSpan(context.Background(), "GET /health")

	s3.End()
}

func TestWrapTracer_ochttp(t *testing.T) {
	defer setupSentryMock(t, []string{
		`{
		  "contexts":{
			"device":"<ignore-diff>",
			"runtime":"<ignore-diff>",
			"trace":{
			  "trace_id":"<ignore-diff>",
			  "span_id":"<ignore-diff>","op":"/do","status":"invalid_argument",
			  "parent_span_id":"<ignore-diff>"
			},
			"os":"<ignore-diff>"
		  },
		  "event_id":"<ignore-diff>",
		  "extra":{
			"http.host":"<ignore-diff>","http.method":"GET","http.path":"/do",
			"http.status_code":400,"http.url":"/do?what",
			"http.user_agent":"Go-http-client/1.1"
		  },
		  "level":"info","platform":"go",
		  "sdk":{
			"name":"sentry.go","version":"0.11.0",
			"integrations":["ContextifyFrames","Environment","IgnoreErrors","Modules"],
			"packages":[{"name":"sentry-go","version":"0.11.0"}]
		  },
		  "server_name":"<ignore-diff>","user":"<ignore-diff>","type":"transaction",
		  "start_timestamp":"<ignore-diff>",
		  "timestamp":"<ignore-diff>","transaction": "client_call"
		}`,
		`{
		  "contexts":{
			"device":"<ignore-diff>",
			"runtime":"<ignore-diff>",
			"trace":{
			  "trace_id":"<ignore-diff>",
			  "span_id":"<ignore-diff>","op":"client_call","status":"invalid_argument"
			},
			"os":"<ignore-diff>"
		  },
		  "event_id":"<ignore-diff>",
		  "extra":{
			"http.host":"<ignore-diff>","http.method":"GET","http.path":"/do",
			"http.status_code":400,"http.url":"<ignore-diff>"
		  },
		  "level":"info","platform":"go",
		  "sdk":{
			"name":"sentry.go","version":"0.11.0",
			"integrations":["ContextifyFrames","Environment","IgnoreErrors","Modules"],
			"packages":[{"name":"sentry-go","version":"0.11.0"}]
		  },
		  "server_name":"<ignore-diff>","user":"<ignore-diff>","type":"transaction",
		  "start_timestamp":"<ignore-diff>",
		  "timestamp":"<ignore-diff>","transaction": "client_call"
		}`,
		`{
		  "contexts":{
			"device":"<ignore-diff>",
			"runtime":"<ignore-diff>",
			"trace":{
			  "trace_id":"<ignore-diff>",
			  "span_id":"<ignore-diff>","op":"client_call","status":"invalid_argument"
			},
			"os":"<ignore-diff>"
		  },
		  "event_id":"<ignore-diff>",
		  "extra":{
			"http.host":"<ignore-diff>","http.method":"GET","http.path":"/do",
			"http.status_code":400,"http.url":"<ignore-diff>"
		  },
		  "level":"info","platform":"go",
		  "sdk":{
			"name":"sentry.go","version":"0.11.0",
			"integrations":["ContextifyFrames","Environment","IgnoreErrors","Modules"],
			"packages":[{"name":"sentry-go","version":"0.11.0"}]
		  },
		  "server_name":"<ignore-diff>","user":"<ignore-diff>","type":"transaction",
		  "start_timestamp":"<ignore-diff>",
		  "timestamp":"<ignore-diff>","transaction": "client_call"
		}`,
	})()

	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("hello!"))
		assert.NoError(t, err)
	})

	h = &ochttp.Handler{Handler: h}
	h = ocsentry.HTTPHandlerMiddleware(h)

	srv := httptest.NewServer(h)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/do?what", nil)
	assert.NoError(t, err)

	resp, err := (&ochttp.Transport{FormatSpanName: func(request *http.Request) string {
		return "client_call"
	}}).RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())

	assert.Equal(t, "hello!", string(b))
}

func BenchmarkWrapTracer(b *testing.B) {
	sentryMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer sentryMock.Close()

	u, err := url.Parse(sentryMock.URL)
	require.NoError(b, err)

	u.User = url.UserPassword("foo", "")
	u.Path = "123123"

	// Enable Sentry wrapper.
	d := trace.DefaultTracer
	trace.DefaultTracer = ocsentry.WrapTracer(trace.DefaultTracer)

	defer func() {
		trace.DefaultTracer = d
	}()

	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.AlwaysSample(),
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i <= b.N; i++ {
		ctx, s1 := trace.StartSpan(context.Background(), "s1")
		_, s2 := trace.StartSpan(ctx, "s2")
		s2.End()
		s1.End()
	}
}

func BenchmarkNoWrapTracer(b *testing.B) {
	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.AlwaysSample(),
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i <= b.N; i++ {
		ctx, s1 := trace.StartSpan(context.Background(), "s1")
		_, s2 := trace.StartSpan(ctx, "s2")
		s2.End()
		s1.End()
	}
}
