package muxtest

import (
	"net/http"
	"net/http/httptest"

	"github.com/cago-frame/cago/pkg/utils/validator"
	"github.com/cago-frame/cago/server/mux"
	"github.com/cago-frame/cago/server/mux/muxclient"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type TestMux struct {
	*mux.Router
	*muxclient.Client
}

type testTransport struct {
	r *gin.Engine
}

func (t *testTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	t.r.ServeHTTP(w, r)
	return w.Result(), nil
}

type Options struct {
	baseUrl string
}

type Option func(*Options)

func newOptions(opts ...Option) *Options {
	options := &Options{
		baseUrl: "/api/v1",
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

func WithBaseUrl(baseUrl string) Option {
	return func(o *Options) {
		o.baseUrl = baseUrl
	}
}

func NewTestMux(opts ...Option) *TestMux {
	options := newOptions(opts...)
	r := gin.Default()
	var err error
	binding.Validator, err = validator.NewValidator()
	if err != nil {
		panic(err)
	}
	// ginContext支持fallback
	r.ContextWithFallback = true
	return &TestMux{
		Router: &mux.Router{
			Routes:  &mux.Routes{IRoutes: r},
			IRouter: r,
		},
		Client: muxclient.NewClient(options.baseUrl, muxclient.WithClient(&http.Client{
			Transport: &testTransport{r: r},
		})),
	}
}
