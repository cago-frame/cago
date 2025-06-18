package muxtest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cago-frame/cago/server/mux"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type testRequest struct {
	mux.Meta `path:"/test" method:"POST"`
	Time     int64 `json:"time"`
}

type testResponse struct {
	Time int64 `json:"time"`
}

func testHandler(ctx context.Context, req *testRequest) (*testResponse, error) {
	return &testResponse{Time: req.Time}, nil
}

type testFormRequest struct {
	mux.Meta `path:"/test/form" method:"POST"`
	Time     int64 `form:"time"`
}

type testFormResponse struct {
	Time int64 `json:"time"`
}

func testFormHandler(ctx context.Context, req *testFormRequest) (*testFormResponse, error) {
	return &testFormResponse{Time: req.Time}, nil
}

type uriRequest struct {
	mux.Meta `path:"/test/uri/:time" method:"GET"`
	Time     int64 `uri:"time"`
}

type uriResponse struct {
	Time int64 `json:"time"`
}

func uriHandler(ctx context.Context, req *uriRequest) (*uriResponse, error) {
	return &uriResponse{Time: req.Time}, nil
}

type queryRequest struct {
	mux.Meta `path:"/test/query" method:"GET"`
	Time     int64 `form:"time"`
}

type queryResponse struct {
	Time int64 `json:"time"`
}

func queryHandler(ctx context.Context, req *queryRequest) (*queryResponse, error) {
	return &queryResponse{Time: req.Time}, nil
}

func TestTestMux(t *testing.T) {
	tr := gin.Default()
	req1, _ := http.NewRequest("GET", "/test/1234", nil)
	w := httptest.NewRecorder()
	tr.GET("/test/:time", func(c *gin.Context) {
		a := c.Param("time")
		c.JSON(200, gin.H{"time": a})
	})
	tr.ServeHTTP(w, req1)

	r := NewTestMux()
	rg := r.Group("/api/v1")

	rg.Bind(testHandler)
	rg.Bind(testFormHandler)
	rg.Bind(uriHandler)
	rg.Bind(queryHandler)

	req := &testRequest{Time: time.Now().Unix()}
	resp := &testResponse{}
	err := r.Do(context.Background(), req, resp)
	assert.NoError(t, err)
	assert.Equal(t, resp.Time, req.Time)

	formReq := &testFormRequest{Time: time.Now().Unix()}
	formResp := &testFormResponse{}
	err = r.Do(context.Background(), formReq, formResp)
	assert.NoError(t, err)
	assert.Equal(t, formResp.Time, formReq.Time)

	uriReq := &uriRequest{Time: time.Now().Unix()}
	uriResp := &uriResponse{}
	err = r.Do(context.Background(), uriReq, uriResp)
	assert.NoError(t, err)
	assert.Equal(t, uriResp.Time, uriReq.Time)

	queryReq := &queryRequest{Time: time.Now().Unix()}
	queryResp := &queryResponse{}
	err = r.Do(context.Background(), queryReq, queryResp)
	assert.NoError(t, err)
	assert.Equal(t, queryResp.Time, queryReq.Time)
}
