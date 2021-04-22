package ocsentry_test

import (
	"context"
	"testing"

	"github.com/vearutop/ocsentry"
)

type errFields map[string]interface{}

func (e errFields) Error() string {
	return "failed"
}

func (e errFields) Fields() map[string]interface{} {
	return e
}

func TestErrorHandler(t *testing.T) {
	defer setupSentryMock(t, []string{`{
        	            	  "contexts":"<ignore-diff>",
        	            	  "event_id":"<ignore-diff>","extra":{"input":123,"foo":1,"bar":2.22},
        	            	  "level":"error","platform":"go",
        	            	  "sdk":"<ignore-diff>",
        	            	  "server_name":"<ignore-diff>","user":"<ignore-diff>",
        	            	  "exception":[
        	            	    {
        	            	      "type":"ocsentry_test.errFields","value":"failed",
        	            	      "stacktrace":{
        	            	        "frames":"<ignore-diff>"
        	            	      }
        	            	    }
        	            	  ],
        	            	  "timestamp":"<ignore-diff>"
        	            	}`})()

	ocsentry.ErrorHandler(context.Background(), 123, errFields{"foo": 1, "bar": 2.22})
}

func TestOnPanic(t *testing.T) {
	defer setupSentryMock(t, []string{`{
        	            	  "contexts":"<ignore-diff>",
        	            	  "event_id":"<ignore-diff>",
        	            	  "level":"fatal","platform":"go",
        	            	  "sdk":"<ignore-diff>",
        	            	  "server_name":"<ignore-diff>","user":"<ignore-diff>",
        	            	  "timestamp":"<ignore-diff>", "message": "oops!"
        	            	}`})()

	ocsentry.OnPanic(context.Background(), "oops!")
}
