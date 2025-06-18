package metric

import (
	"net/http"
	"time"

	"github.com/cago-frame/cago"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const instrumName = "github.com/cago-frame/cago/pkg/opentelemetry/metric.http"

func Middleware(m metric.MeterProvider) (gin.HandlerFunc, error) {
	attr := []metric.MeterOption{
		metric.WithInstrumentationVersion(cago.Version()),
	}
	requestTotal, err := m.Meter(instrumName, attr...).Int64Counter(
		"http_request_total", metric.WithDescription("http请求数量"),
	)
	if err != nil {
		return nil, err
	}
	requestDuration, err := m.Meter(instrumName, attr...).Int64Counter(
		"http_request_duration", metric.WithDescription("http请求耗时"),
	)
	if err != nil {
		return nil, err
	}
	durationBucket := []int64{100, 300, 500, 1000, 2000, 5000, 10000}

	requestBodySize, err := m.Meter(instrumName, attr...).Int64Counter(
		"http_request_body_size", metric.WithDescription("http请求body大小"),
	)
	if err != nil {
		return nil, err
	}
	responseBodySize, err := m.Meter(instrumName, attr...).Int64Counter(
		"http_response_body_size", metric.WithDescription("http响应body大小"),
	)
	if err != nil {
		return nil, err
	}
	httpStatusCode, err := m.Meter(instrumName, attr...).Int64Counter(
		"http_status_code", metric.WithDescription("http响应状态码"),
	)
	if err != nil {
		return nil, err
	}

	return func(c *gin.Context) {
		ts := time.Now()
		fullPath := "/"
		if c.FullPath() != "" {
			fullPath = c.FullPath()
		}
		attr := metric.WithAttributes(
			attribute.String("uri", fullPath),
		)
		requestTotal.Add(c.Request.Context(), 1,
			attr,
		)
		requestBodySize.Add(c.Request.Context(), c.Request.ContentLength,
			attr,
		)

		c.Next()

		end := time.Now()
		for _, v := range durationBucket {
			if end.Sub(ts).Milliseconds() < int64(v) {
				requestDuration.Add(c.Request.Context(), 1,
					attr,
					metric.WithAttributes(
						attribute.Int64("bucket", v),
					),
				)
				break
			}
		}

		responseBodySize.Add(c.Request.Context(), int64(c.Writer.Size()),
			attr,
		)
		code := http.StatusOK
		if c.Writer.Status() != 0 {
			code = c.Writer.Status()
		}
		httpStatusCode.Add(c.Request.Context(), 1,
			attr,
			metric.WithAttributes(
				attribute.Int("status_code", code)),
		)
	}, nil
}
