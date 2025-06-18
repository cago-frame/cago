package mux

import (
	"context"
	"testing"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/configs/memory"
)

type testRequest struct {
	Meta `path:"/test" method:"GET"`
	Time int64 `json:"time"`
}

type testResponse struct {
	Time int64 `json:"time"`
}

func TestMux(t *testing.T) {
	cfg, err := configs.NewConfig("http-test", configs.WithSource(memory.NewSource(map[string]interface{}{})))
	if err != nil {
		t.Fatal("failed to create config: ", err)
	}

	_ = HTTP(func(ctx context.Context, r *Router) error {
		r.Bind(func(ctx context.Context, req *testRequest) (*testResponse, error) {
			return &testResponse{Time: req.Time}, nil
		})
		return nil
	}).Start(context.Background(), cfg)

}
