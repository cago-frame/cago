package muxclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/cago-frame/cago/pkg/utils"
	"github.com/cago-frame/cago/server/mux"

	"github.com/cago-frame/cago/pkg/utils/httputils"
)

type ClientOptions struct {
	client *http.Client
}

type ClientOption func(*ClientOptions)

func WithClient(client *http.Client) ClientOption {
	return func(options *ClientOptions) {
		options.client = client
	}
}

type ClientDoOptions struct {
	path   string
	resp   **http.Response
	header http.Header
}

type ClientDoOption func(*ClientDoOptions)

func newDoOptions(opts ...ClientDoOption) *ClientDoOptions {
	options := &ClientDoOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

func WithPath(path string) ClientDoOption {
	return func(options *ClientDoOptions) {
		options.path = path
	}
}

func WithResponse(resp **http.Response) ClientDoOption {
	return func(options *ClientDoOptions) {
		options.resp = resp
	}
}

func WithHeader(header http.Header) ClientDoOption {
	return func(options *ClientDoOptions) {
		options.header = header
	}
}

type Client struct {
	options *ClientOptions
	baseURL string
}

func NewClient(baseURL string, opts ...ClientOption) *Client {
	options := &ClientOptions{client: http.DefaultClient}
	for _, opt := range opts {
		opt(options)
	}
	return &Client{
		options: options,
		baseURL: baseURL,
	}
}

func (c *Client) Request(ctx context.Context, req any, opts ...ClientDoOption) (*http.Request, error) {
	options := newDoOptions(opts...)
	route, ok := reflect.TypeOf(req).Elem().FieldByName("Meta")
	if !ok {
		return nil, errors.New("invalid method, second parameter must have Meta field")
	}
	// 必须有Route字段
	if route.Type != reflect.TypeOf(mux.Meta{}) {
		return nil, errors.New("invalid method, second parameter must have Meta field")
	}
	method := route.Tag.Get("method")
	path := route.Tag.Get("path")
	if options.path != "" {
		path = options.path
	}

	ptrVal := reflect.ValueOf(req)
	ptrElem := ptrVal.Elem()
	ptrType := ptrElem.Type()

	query := url.Values{}
	data := make(map[string]interface{})
	var form func(key string, value any)
	switch method {
	case http.MethodGet, http.MethodDelete:
		form = func(key string, value any) {
			switch value := value.(type) {
			case string:
				query.Add(key, value)
			case bool:
				query.Add(key, strconv.FormatBool(value))
			case int8, int16, int, int32, int64, uint8, uint16, uint, uint32, uint64:
				query.Add(key, fmt.Sprintf("%d", value))
			case float32, float64:
				query.Add(key, fmt.Sprintf("%f", value))
			}
		}
	default:
		form = func(key string, value any) {
			data[key] = value
		}
	}

	for i := 0; i < ptrElem.NumField(); i++ {
		tag := ptrType.Field(i).Tag
		if tag == "" {
			continue
		}
		fieldElem := ptrElem.Field(i)
		key := tag.Get("json")
		if key == "" {
			key = tag.Get("form")
		}
		if key != "" {
			if key == "-" {
				continue
			}
			if key == ",inline" {
				data[key] = fieldElem.Interface()
			} else {
				// 处理key,label的情况,例如: key,default=1
				key, opts := utils.Head(key, ",")
				opts, val := utils.Head(opts, "=")
				if opts == "default" && fieldElem.IsZero() {
					form(key, val)
				} else {
					form(key, fieldElem.Interface())
				}
			}
		} else if uri := tag.Get("uri"); uri != "" {
			path = strings.ReplaceAll(path, ":"+uri, fmt.Sprintf("%v", fieldElem.Interface()))
		}
	}
	var body io.Reader
	if len(data) != 0 {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(b)
	}

	path = c.baseURL + path
	if len(query) != 0 {
		path += "?" + query.Encode()
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	if options.header != nil {
		for k, v := range options.header {
			httpReq.Header[k] = v
		}
	}
	if body != nil {
		if httpReq.Header.Get("Content-Type") == "" {
			httpReq.Header.Set("Content-Type", "application/json")
		}
	}
	return httpReq, nil
}

func (c *Client) Do(ctx context.Context, req any, resp any, opts ...ClientDoOption) error {
	httpReq, err := c.Request(ctx, req, opts...)
	if err != nil {
		return err
	}
	return c.HttpDo(httpReq, resp, opts...)
}

func (c *Client) HttpDo(httpReq *http.Request, resp any, opts ...ClientDoOption) error {
	options := newDoOptions(opts...)
	httpResp, err := c.options.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() {
		_ = httpResp.Body.Close()
	}()
	b, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}
	jsonResp := &httputils.JSONResponse{
		Data: resp,
	}
	if err := json.Unmarshal(b, jsonResp); err != nil {
		return fmt.Errorf("json unmarshal error: %w", err)
	}
	if jsonResp.Code != 0 {
		return httputils.NewError(httpResp.StatusCode, jsonResp.Code, jsonResp.Msg)
	}
	if options.resp != nil {
		*options.resp = httpResp
	}
	return nil
}
