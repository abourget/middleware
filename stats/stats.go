package stats

import (
	"strconv"

	"github.com/armon/go-metrics"
	"github.com/goadesign/goa"
)

// ResponseReporter is a middleware that reports response code and error statistics to any sink
// supported by github.com/armon/go-metrics, which currently
// includes statsd, prometheus and others.
func ResponseReporter(sink metrics.MetricSink, controllerName string) goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx *goa.Context) error {
			err := h(ctx)
			keys := []string{controllerName, ctx.Request().Method, strconv.Itoa(ctx.ResponseStatus())}
			increment(keys, 1)
			if err != nil {
				keys := []string{controllerName, ctx.Request().Method, "errors"}
				increment(keys, 1)
			}

			return err
		}
	}

}

func increment(keys []string, count int) {
	metrics.IncrCounter(keys, float32(count))

}
