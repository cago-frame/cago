package oss

import (
	"context"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/cago-frame/cago/pkg/oss/oss"
	"github.com/cago-frame/cago/pkg/utils/wrap"
	"github.com/stretchr/testify/assert"
)

type emptyClient struct {
	oss.Client
}

type emptyBucket struct {
	oss.Bucket
}

func (c *emptyClient) ListBuckets(ctx context.Context) ([]*oss.BucketInfo, error) {
	return nil, nil
}

func (c *emptyClient) Bucket(ctx context.Context, name string) (oss.Bucket, error) {
	return &emptyBucket{}, nil
}

func (b *emptyBucket) PutObject(ctx context.Context, objectName string, data io.Reader, objectSize int64) error {
	return nil
}

func (b *emptyBucket) PreSignedPutObject(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error) {
	return nil, nil
}

func (b *emptyBucket) GetObject(ctx context.Context, objectName string) (oss.Object, error) {
	return nil, nil
}

func (b *emptyBucket) PreSignedGetObject(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error) {
	return nil, nil
}

func (b *emptyBucket) RemoveObject(ctx context.Context, objectName string) error {
	return nil
}

func Test_newWrapClient(t *testing.T) {
	w := wrap.New()
	// 计数器变量
	var listBucketCallCount int

	w.Wrap(func(ctx *wrap.Context) {
		switch ctx.Name() {
		case "ListBuckets":
			listBucketCallCount++
		}
	})
	c := newWrapClient(&emptyClient{}, w)
	_, _ = c.ListBuckets(context.Background())

	// 断言调用计数
	assert.Equal(t, listBucketCallCount, 1)
}

func Test_newWrapBucket(t *testing.T) {
	w := wrap.New()
	// 计数器变量
	var (
		putObjectCallCount          int
		preSignedPutObjectCallCount int
		getObjectCallCount          int
		preSignedGetObjectCallCount int
		removeObjectCallCount       int
	)

	w.Wrap(func(ctx *wrap.Context) {
		switch ctx.Name() {
		case "PutObject":
			putObjectCallCount++
			assert.Equal(t, ctx.Args(0), "object-name")
			assert.Nil(t, ctx.Args(1)) // 传入的数据为nil
		case "PreSignedPutObject":
			preSignedPutObjectCallCount++
			assert.Equal(t, ctx.Args(0), "object-name")
			assert.Equal(t, ctx.Args(1), time.Minute)
		case "GetObject":
			getObjectCallCount++
			assert.Equal(t, ctx.Args(0), "object-name")
		case "PreSignedGetObject":
			preSignedGetObjectCallCount++
			assert.Equal(t, ctx.Args(0), "object-name")
			assert.Equal(t, ctx.Args(1), time.Minute)
		case "RemoveObject":
			removeObjectCallCount++
			assert.Equal(t, ctx.Args(0), "object-name")
		}
	})
	b := newWrapBucket(&emptyBucket{}, w)
	_ = b.PutObject(context.Background(), "object-name", nil, 0)
	_, _ = b.PreSignedPutObject(context.Background(), "object-name", time.Minute)
	_, _ = b.GetObject(context.Background(), "object-name")
	_, _ = b.PreSignedGetObject(context.Background(), "object-name", time.Minute)
	_ = b.RemoveObject(context.Background(), "object-name")

	// 断言调用计数
	assert.Equal(t, putObjectCallCount, 1)
	assert.Equal(t, preSignedPutObjectCallCount, 1)
	assert.Equal(t, getObjectCallCount, 1)
	assert.Equal(t, preSignedGetObjectCallCount, 1)
	assert.Equal(t, removeObjectCallCount, 1)
}
