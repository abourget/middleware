package middleware_test

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/goadesign/goa"
	"github.com/goadesign/middleware"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewMiddleware", func() {
	var input interface{}
	var middleware goa.Middleware
	var mErr error

	JustBeforeEach(func() {
		middleware, mErr = goa.NewMiddleware(input)
	})

	Context("using a goa Middleware", func() {
		var goaMiddleware goa.Middleware

		BeforeEach(func() {
			goaMiddleware = func(h goa.Handler) goa.Handler { return h }
			input = goaMiddleware
		})

		It("returns the middleware", func() {
			Ω(fmt.Sprintf("%#v", middleware)).Should(Equal(fmt.Sprintf("%#v", goaMiddleware)))
			Ω(mErr).ShouldNot(HaveOccurred())
		})
	})

	Context("using a goa middleware func", func() {
		var goaMiddlewareFunc func(goa.Handler) goa.Handler

		BeforeEach(func() {
			goaMiddlewareFunc = func(h goa.Handler) goa.Handler { return h }
			input = goaMiddlewareFunc
		})

		It("returns the middleware", func() {
			Ω(fmt.Sprintf("%#v", middleware)).Should(Equal(fmt.Sprintf("%#v", goa.Middleware(goaMiddlewareFunc))))
			Ω(mErr).ShouldNot(HaveOccurred())
		})
	})

	Context("with a context", func() {
		var service goa.Service
		var ctx context.Context
		var req *http.Request
		var rw http.ResponseWriter
		var params url.Values

		BeforeEach(func() {
			service = goa.New("test")
			service.SetEncoder(goa.JSONEncoderFactory(), true, "*/*")
			var err error
			req, err = http.NewRequest("GET", "/goo", nil)
			Ω(err).ShouldNot(HaveOccurred())
			rw = new(testResponseWriter)
			params = url.Values{"query": []string{"value"}}
			ctx = goa.NewContext(nil, service, rw, req, params)
			Ω(goa.Response(ctx).Status).Should(Equal(0))
		})

		Context("using a goa handler", func() {
			BeforeEach(func() {
				var goaHandler goa.Handler = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
					return goa.Response(ctx).Send(ctx, 200, "ok")
				}
				input = goaHandler
			})

			It("wraps it in a middleware", func() {
				Ω(mErr).ShouldNot(HaveOccurred())
				h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error { return nil }
				Ω(middleware(h)(ctx, rw, req)).ShouldNot(HaveOccurred())
				Ω(goa.Response(ctx).Status).Should(Equal(200))
			})
		})

		Context("using a goa handler func", func() {
			BeforeEach(func() {
				input = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
					return goa.Response(ctx).Send(ctx, 200, "ok")
				}
			})

			It("wraps it in a middleware", func() {
				Ω(mErr).ShouldNot(HaveOccurred())
				h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error { return nil }
				Ω(middleware(h)(ctx, rw, req)).ShouldNot(HaveOccurred())
				Ω(goa.Response(ctx).Status).Should(Equal(200))
			})
		})

		Context("using a http middleware func", func() {
			BeforeEach(func() {
				input = func(h http.Handler) http.Handler { return h }
			})

			It("wraps it in a middleware", func() {
				Ω(mErr).ShouldNot(HaveOccurred())
				h := func(c context.Context, rw http.ResponseWriter, req *http.Request) error {
					return goa.Response(ctx).Send(ctx, 200, "ok")
				}
				Ω(middleware(h)(ctx, rw, req)).ShouldNot(HaveOccurred())
				Ω(goa.Response(ctx).Status).Should(Equal(200))
			})
		})

		Context("using a http handler", func() {
			BeforeEach(func() {
				var httpHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("ok"))
					w.WriteHeader(200)
				})
				input = httpHandler
			})

			It("wraps it in a middleware", func() {
				Ω(mErr).ShouldNot(HaveOccurred())
				h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error { return nil }
				Ω(middleware(h)(ctx, rw, req)).ShouldNot(HaveOccurred())
				Ω(rw.(*testResponseWriter).Status).Should(Equal(200))
			})
		})

		Context("using a http handler func", func() {
			BeforeEach(func() {
				input = func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("ok"))
					w.WriteHeader(200)
				}
			})

			It("wraps it in a middleware", func() {
				Ω(mErr).ShouldNot(HaveOccurred())
				var newCtx context.Context
				h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
					newCtx = ctx
					return nil
				}
				Ω(middleware(h)(ctx, rw, req)).ShouldNot(HaveOccurred())
				Ω(rw.(*testResponseWriter).Status).Should(Equal(200))
			})
		})

	})
})

var _ = Describe("LogRequest", func() {
	var ctx context.Context
	var rw http.ResponseWriter
	var req *http.Request
	var params url.Values
	var logger *testLogger

	payload := map[string]interface{}{"payload": 42}

	BeforeEach(func() {
		service := goa.New("test")
		service.SetEncoder(goa.JSONEncoderFactory(), true, "*/*")
		var err error
		req, err = http.NewRequest("POST", "/goo?param=value", strings.NewReader(`{"payload":42}`))
		Ω(err).ShouldNot(HaveOccurred())
		rw = new(testResponseWriter)
		params = url.Values{"query": []string{"value"}}
		ctx = goa.NewContext(nil, service, rw, req, params)
		goa.Request(ctx).Payload = payload
		logger = new(testLogger)
		goa.Log = logger
	})

	It("logs requests", func() {
		h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			return goa.Response(ctx).Send(ctx, 200, "ok")
		}
		lg := middleware.LogRequest(true)(h)
		Ω(lg(ctx, rw, req)).ShouldNot(HaveOccurred())
		Ω(logger.InfoEntries).Should(HaveLen(4))

		Ω(logger.InfoEntries[0].Data).Should(HaveLen(2))
		Ω(logger.InfoEntries[0].Data[0].Key).Should(Equal("id"))
		Ω(logger.InfoEntries[0].Data[1].Key).Should(Equal("POST"))
		Ω(logger.InfoEntries[0].Data[1].Value).Should(Equal("/goo?param=value"))

		Ω(logger.InfoEntries[1].Data).Should(HaveLen(2))
		Ω(logger.InfoEntries[0].Data[0].Key).Should(Equal("id"))
		Ω(logger.InfoEntries[1].Data[1].Key).Should(Equal("query"))
		Ω(logger.InfoEntries[1].Data[1].Value).Should(Equal("value"))

		Ω(logger.InfoEntries[2].Data).Should(HaveLen(2))
		Ω(logger.InfoEntries[0].Data[0].Key).Should(Equal("id"))
		Ω(logger.InfoEntries[2].Data[1].Key).Should(Equal("payload"))
		Ω(logger.InfoEntries[2].Data[1].Value).Should(Equal(42))

		Ω(logger.InfoEntries[3].Data).Should(HaveLen(4))
		Ω(logger.InfoEntries[0].Data[0].Key).Should(Equal("id"))
		Ω(logger.InfoEntries[3].Data[1].Key).Should(Equal("status"))
		Ω(logger.InfoEntries[3].Data[2].Key).Should(Equal("bytes"))
		Ω(logger.InfoEntries[3].Data[1].Value).Should(Equal(200))
		Ω(logger.InfoEntries[3].Data[2].Value).Should(Equal(5))
		Ω(logger.InfoEntries[3].Data[3].Key).Should(Equal("time"))
	})
})

var _ = Describe("LogResponse", func() {
	var logger *testLogger
	var ctx context.Context
	var req *http.Request
	var rw http.ResponseWriter
	var params url.Values
	responseText := "some response data to be logged"

	BeforeEach(func() {
		var err error
		req, err = http.NewRequest("POST", "/goo", strings.NewReader(`{"payload":42}`))
		Ω(err).ShouldNot(HaveOccurred())
		rw = new(testResponseWriter)
		params = url.Values{"query": []string{"value"}}
		ctx = goa.NewContext(nil, goa.New("test"), rw, req, params)
		logger = new(testLogger)
		goa.Log = logger
	})

	It("logs responses", func() {
		h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			goa.Response(ctx).WriteHeader(200)
			goa.Response(ctx).Write([]byte(responseText))
			return nil
		}
		lg := middleware.LogResponse()(h)
		Ω(lg(ctx, rw, req)).ShouldNot(HaveOccurred())
		Ω(logger.InfoEntries).Should(HaveLen(1))

		Ω(logger.InfoEntries[0].Data).Should(HaveLen(1))
		Ω(logger.InfoEntries[0].Data[0].Key).Should(Equal("raw"))
		Ω(logger.InfoEntries[0].Data[0].Value).Should(Equal(responseText))
	})
})

var _ = Describe("RequestID", func() {
	const reqID = "request id"
	var ctx context.Context
	var rw http.ResponseWriter
	var req *http.Request
	var params url.Values

	BeforeEach(func() {
		var err error
		req, err = http.NewRequest("GET", "/goo", nil)
		Ω(err).ShouldNot(HaveOccurred())
		req.Header.Set("X-Request-Id", reqID)
		rw = new(testResponseWriter)
		service := goa.New("test")
		params = url.Values{"query": []string{"value"}}
		service.SetEncoder(goa.JSONEncoderFactory(), true, "*/*")
		ctx = goa.NewContext(nil, service, rw, req, params)
	})

	It("sets the request ID in the context", func() {
		var newCtx context.Context
		h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			newCtx = ctx
			return goa.Response(ctx).Send(ctx, 200, "ok")
		}
		rg := middleware.RequestID()(h)
		Ω(rg(ctx, rw, req)).ShouldNot(HaveOccurred())
		Ω(newCtx.Value(middleware.ReqIDKey)).Should(Equal(reqID))
	})
})

var _ = Describe("Recover", func() {
	It("recovers", func() {
		goa.Log = nil
		h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			panic("boom")
		}
		rg := middleware.Recover()(h)
		service := goa.New("test")
		service.SetEncoder(goa.JSONEncoderFactory(), true, "*/*")
		rw := new(testResponseWriter)
		err := rg(goa.NewContext(nil, service, rw, nil, nil), rw, nil)
		Ω(err).Should(HaveOccurred())
		Ω(err.Error()).Should(Equal("panic: boom"))
	})
})

var _ = Describe("Timeout", func() {
	It("sets a deadline", func() {
		req, err := http.NewRequest("POST", "/goo", strings.NewReader(`{"payload":42}`))
		Ω(err).ShouldNot(HaveOccurred())
		rw := new(testResponseWriter)
		service := goa.New("test")
		service.SetEncoder(goa.JSONEncoderFactory(), true, "*/*")

		ctx := goa.NewContext(nil, service, rw, req, nil)
		var newCtx context.Context
		h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			newCtx = ctx
			return goa.Response(ctx).Send(ctx, 200, "ok")
		}
		t := middleware.Timeout(time.Duration(1))(h)
		err = t(ctx, rw, req)
		Ω(err).ShouldNot(HaveOccurred())
		_, ok := newCtx.Deadline()
		Ω(ok).Should(BeTrue())
	})
})

var _ = Describe("RequireHeader", func() {
	var ctx context.Context
	var req *http.Request
	var rw http.ResponseWriter
	headerName := "Some-Header"

	BeforeEach(func() {
		var err error
		service := goa.New("test")
		service.SetEncoder(goa.JSONEncoderFactory(), true, "*/*")
		req, err = http.NewRequest("POST", "/foo/bar", strings.NewReader(`{"payload":42}`))
		Ω(err).ShouldNot(HaveOccurred())
		rw = new(testResponseWriter)
		ctx = goa.NewContext(nil, service, rw, req, nil)
	})

	It("matches a header value", func() {
		req.Header.Set(headerName, "some value")
		var newCtx context.Context
		h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			newCtx = ctx
			return goa.Response(ctx).Send(ctx, http.StatusOK, "ok")
		}
		t := middleware.RequireHeader(
			regexp.MustCompile("^/foo"),
			headerName,
			regexp.MustCompile("^some value$"),
			http.StatusUnauthorized)(h)
		err := t(ctx, rw, req)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(goa.Response(newCtx).Status).Should(Equal(http.StatusOK))
	})

	It("responds with failure on mismatch", func() {
		req.Header.Set(headerName, "some other value")
		h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			panic("unreachable")
		}
		t := middleware.RequireHeader(
			regexp.MustCompile("^/foo"),
			headerName,
			regexp.MustCompile("^some value$"),
			http.StatusUnauthorized)(h)
		err := t(ctx, rw, req)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(goa.Response(ctx).Status).Should(Equal(http.StatusUnauthorized))
	})

	It("responds with failure when header is missing", func() {
		h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			panic("unreachable")
		}
		t := middleware.RequireHeader(
			regexp.MustCompile("^/foo"),
			headerName,
			regexp.MustCompile("^some value$"),
			http.StatusUnauthorized)(h)
		err := t(ctx, rw, req)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(goa.Response(ctx).Status).Should(Equal(http.StatusUnauthorized))
	})

	It("passes through for a non-matching path", func() {
		var newCtx context.Context
		req.Header.Set(headerName, "bogus")
		h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			newCtx = ctx
			return goa.Response(ctx).Send(ctx, http.StatusOK, "ok")
		}
		t := middleware.RequireHeader(
			regexp.MustCompile("^/baz"),
			headerName,
			regexp.MustCompile("^some value$"),
			http.StatusUnauthorized)(h)
		err := t(ctx, rw, req)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(goa.Response(newCtx).Status).Should(Equal(http.StatusOK))
	})

	It("matches value for a nil path pattern", func() {
		req.Header.Set(headerName, "bogus")
		h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			panic("unreachable")
		}
		t := middleware.RequireHeader(
			nil,
			headerName,
			regexp.MustCompile("^some value$"),
			http.StatusNotFound)(h)
		err := t(ctx, rw, req)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(goa.Response(ctx).Status).Should(Equal(http.StatusNotFound))
	})
})

type logEntry struct {
	Msg  string
	Data []goa.KV
}

type testLogger struct {
	InfoEntries  []logEntry
	ErrorEntries []logEntry
}

func (t *testLogger) Info(ctx context.Context, msg string, data ...goa.KV) {
	e := logEntry{msg, data}
	t.InfoEntries = append(t.InfoEntries, e)
}

func (t *testLogger) Error(ctx context.Context, msg string, data ...goa.KV) {
	e := logEntry{msg, data}
	t.ErrorEntries = append(t.ErrorEntries, e)
}

type testResponseWriter struct {
	ParentHeader http.Header
	Body         []byte
	Status       int
}

func (t *testResponseWriter) Header() http.Header {
	return t.ParentHeader
}

func (t *testResponseWriter) Write(b []byte) (int, error) {
	t.Body = append(t.Body, b...)
	return len(b), nil
}

func (t *testResponseWriter) WriteHeader(s int) {
	t.Status = s
}
