package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/goadesign/goa"

	log "gopkg.in/inconshreveable/log15.v2"

	"golang.org/x/net/context"
)

// ReqIDKey is the context key used by the RequestID middleware to store the request ID value.
const ReqIDKey middlewareKey = 1

// RequestIDHeader is the name of the header used to transmit the request ID.
const RequestIDHeader = "X-Request-Id"

// Counter used to create new request ids.
var reqID int64

// Common prefix to all newly created request ids for this process.
var reqPrefix string

// Initialize common prefix on process startup.
func init() {
	// algorithm taken from https://github.com/zenazn/goji/blob/master/web/middleware/request_id.go#L44-L50
	var buf [12]byte
	var b64 string
	for len(b64) < 10 {
		rand.Read(buf[:])
		b64 = base64.StdEncoding.EncodeToString(buf[:])
		b64 = strings.NewReplacer("+", "", "/", "").Replace(b64)
	}
	reqPrefix = string(b64[0:10])
}

// middlewareKey is the private type used for goa middlewares to store values in the context.
// It is private to avoid possible collisions with keys used by other packages.
type middlewareKey int

// LogRequest creates a request logger middleware.
// This middleware is aware of the RequestID middleware and if registered after it leverages the
// request ID for logging.
func LogRequest() goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx *goa.Context) error {
			reqID := ctx.Value(ReqIDKey)
			if reqID == nil {
				reqID = shortID()
			}
			ctx.Logger = ctx.Logger.New("id", reqID)
			startedAt := time.Now()
			r := ctx.Request()
			ctx.Info("started", r.Method, r.URL.String())
			params := ctx.AllParams()
			if len(params) > 0 {
				logCtx := make(log.Ctx, len(params))
				for k, v := range params {
					logCtx[k] = interface{}(v)
				}
				ctx.Debug("params", logCtx)
			}
			payload := ctx.RawPayload()
			if r.ContentLength > 0 {
				if mp, ok := payload.(map[string]interface{}); ok {
					ctx.Debug("payload", log.Ctx(mp))
				} else {
					ctx.Debug("payload", "raw", payload)
				}
			}
			err := h(ctx)
			ctx.Info("completed", "status", ctx.ResponseStatus(),
				"bytes", ctx.ResponseLength(), "time", time.Since(startedAt).String())
			return err
		}
	}
}

// loggingResponseWriter wraps an http.ResponseWriter and writes only raw
// response data (as text) to the context logger. assumes status and duration
// are logged elsewhere (i.e. by the LogRequest middleware).
type loggingResponseWriter struct {
	http.ResponseWriter
	ctx *goa.Context
}

// Write will write raw data to logger and response writer.
func (lrw *loggingResponseWriter) Write(buf []byte) (int, error) {
	lrw.ctx.Logger.Debug("response", "raw", string(buf))
	return lrw.ResponseWriter.Write(buf)
}

// LogResponse creates a response logger middleware.
// Only Logs the raw response data without accumulating any statistics.
func LogResponse() goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx *goa.Context) error {
			// chain a new logging writer to the current response writer.
			ctx.SetResponseWriter(
				&loggingResponseWriter{
					ResponseWriter: ctx.SetResponseWriter(nil),
					ctx:            ctx})

			// next
			return h(ctx)
		}
	}
}

// RequestID is a middleware that injects a request ID into the context of each request.
// Retrieve it using ctx.Value(ReqIDKey). If the incoming request has a RequestIDHeader header then
// that value is used else a random value is generated.
func RequestID() goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx *goa.Context) error {
			id := ctx.Request().Header.Get(RequestIDHeader)
			if id == "" {
				id = fmt.Sprintf("%s-%d", reqPrefix, atomic.AddInt64(&reqID, 1))
			}
			ctx.SetValue(ReqIDKey, id)

			return h(ctx)
		}
	}
}

// Recover is a middleware that recovers panics and returns an internal error response.
func Recover() goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx *goa.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					if ctx != nil {
						switch x := r.(type) {
						case string:
							err = fmt.Errorf("panic: %s", x)
						case error:
							err = x
						default:
							err = errors.New("unknown panic")
						}
						const size = 64 << 10 // 64KB
						buf := make([]byte, size)
						buf = buf[:runtime.Stack(buf, false)]
						lines := strings.Split(string(buf), "\n")
						stack := lines[3:]
						status := http.StatusInternalServerError
						var message string
						if ctx.Logger != nil {
							reqID := ctx.Value(ReqIDKey)
							if reqID != nil {
								message = fmt.Sprintf(
									"%s\nRefer to the following token when contacting support: %s",
									http.StatusText(status),
									reqID)
							}
							ctx.Logger.Error("panic", "err", err, "stack", stack)
						}

						// note we must respond or else a 500 with "unhandled request" is the
						// default response.
						if message == "" {
							// without the logger and/or request id (from middleware) we can
							// only return the full error message for reference purposes. it
							// is unlikely to make sense to the caller unless they understand
							// the source code.
							message = err.Error()
						}
						ctx.RespondBytes(status, []byte(message))
					}
				}
			}()
			return h(ctx)
		}
	}
}

// Timeout sets a global timeout for all controller actions.
// The timeout notification is made through the context, it is the responsability of the request
// handler to handle it. For example:
//
// 	func (ctrl *Controller) DoLongRunningAction(ctx *DoLongRunningActionContext) error {
// 		action := NewLongRunning()      // setup long running action
//		c := make(chan error, 1)        // create return channel
//		go func() { c <- action.Run() } // Launch long running action goroutine
//		select {
//		case <- ctx.Done():             // timeout triggered
//			action.Cancel()         // cancel long running action
//			<-c                     // wait for Run to return.
//			return ctx.Err()        // retrieve cancel reason
//		case err := <-c:   		// action finished on time
//			return err  		// forward its return value
//		}
//	}
//
// Package golang.org/x/net/context/ctxhttp contains an implementation of an HTTP client which is
// context-aware:
//
// 	func (ctrl *Controller) HttpAction(ctx *HttpActionContext) error {
//		req, err := http.NewRequest("GET", "http://iamaslowservice.com", nil)
//		// ...
//		resp, err := ctxhttp.Do(ctx, nil, req) // returns if timeout triggers
//		// ...
// 	}
//
// Controller actions can check if a timeout is set by calling the context Deadline method.
func Timeout(timeout time.Duration) goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx *goa.Context) (err error) {
			// We discard the cancel function because the goa handler already takes
			// care of canceling on completion.
			ctx.Context, _ = context.WithTimeout(ctx.Context, timeout)
			return h(ctx)
		}
	}
}

// RequireHeader requires a request header to match a value pattern. If the
// header is missing or does not match then the failureStatus is the response
// (e.g. http.StatusUnauthorized). If pathPattern is nil then any path is
// included. If requiredHeaderValue is nil then any value is accepted so long as
// the header is non-empty.
func RequireHeader(
	pathPattern *regexp.Regexp,
	requiredHeaderName string,
	requiredHeaderValue *regexp.Regexp,
	failureStatus int) goa.Middleware {

	return func(h goa.Handler) goa.Handler {
		return func(ctx *goa.Context) (err error) {
			if pathPattern == nil || pathPattern.MatchString(ctx.Request().URL.Path) {
				matched := false
				header := ctx.Request().Header
				headerValue := header.Get(requiredHeaderName)
				if len(headerValue) > 0 {
					if requiredHeaderValue == nil {
						matched = true
					} else {
						matched = requiredHeaderValue.MatchString(headerValue)
					}
				}
				if matched {
					err = h(ctx)
				} else {
					err = ctx.RespondBytes(failureStatus, []byte(http.StatusText(failureStatus)))
				}
			} else {
				err = h(ctx)
			}
			return
		}
	}
}

// shortID produces a "unique" 6 bytes long string.
// Do not use as a reliable way to get unique IDs, instead use for things like logging.
func shortID() string {
	b := make([]byte, 6)
	io.ReadFull(rand.Reader, b)
	return base64.StdEncoding.EncodeToString(b)
}
