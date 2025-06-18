package oss

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/cago-frame/cago/pkg/oss/oss"
	"github.com/cago-frame/cago/pkg/utils/wrap"
)

type wrapClient struct {
	oss.Client
	wrap *wrap.Wrap
}

func newWrapClient(client oss.Client, w *wrap.Wrap) oss.Client {
	return &wrapClient{
		Client: client,
		wrap:   w,
	}
}

func (t *wrapClient) ListBuckets(ctx context.Context) (list []*oss.BucketInfo, err error) {
	err = t.wrap.Run(ctx, "ListBuckets", []interface{}{}, func(ctx *wrap.Context) {
		list, err = t.Client.ListBuckets(ctx)
		ctx.Abort(err)
	})
	return
}

func (t *wrapClient) Bucket(ctx context.Context, name string) (oss.Bucket, error) {
	bucket, err := t.Client.Bucket(ctx, name)
	if err != nil {
		return nil, err
	}
	return newWrapBucket(bucket, t.wrap), nil
}

type wrapBucket struct {
	oss.Bucket
	wrap *wrap.Wrap
}

func newWrapBucket(bucket oss.Bucket, w *wrap.Wrap) oss.Bucket {
	return &wrapBucket{
		Bucket: bucket,
		wrap:   w,
	}
}

func (t *wrapBucket) PutObject(ctx context.Context, objectName string, data io.Reader, objectSize int64) error {
	return t.wrap.Run(ctx, "PutObject", []interface{}{objectName, data}, func(ctx *wrap.Context) {
		ctx.Abort(t.Bucket.PutObject(ctx, objectName, data, objectSize))
	})
}

func (t *wrapBucket) PreSignedPutObject(ctx context.Context, objectName string, expires time.Duration) (u *url.URL, err error) {
	err = t.wrap.Run(ctx, "PreSignedPutObject", []interface{}{objectName, expires}, func(ctx *wrap.Context) {
		u, err = t.Bucket.PreSignedPutObject(ctx, objectName, expires)
		ctx.Abort(err)
	})
	return
}

func (t *wrapBucket) GetObject(ctx context.Context, objectName string) (object oss.Object, err error) {
	err = t.wrap.Run(ctx, "GetObject", []interface{}{objectName}, func(ctx *wrap.Context) {
		object, err = t.Bucket.GetObject(ctx, objectName)
		ctx.Abort(err)
	})
	return
}

func (t *wrapBucket) PreSignedGetObject(ctx context.Context, objectName string, expires time.Duration) (u *url.URL, err error) {
	err = t.wrap.Run(ctx, "PreSignedGetObject", []interface{}{objectName, expires}, func(ctx *wrap.Context) {
		u, err = t.Bucket.PreSignedGetObject(ctx, objectName, expires)
		ctx.Abort(err)
	})
	return
}

func (t *wrapBucket) RemoveObject(ctx context.Context, objectName string) error {
	return t.wrap.Run(ctx, "RemoveObject", []interface{}{objectName}, func(ctx *wrap.Context) {
		ctx.Abort(t.Bucket.RemoveObject(ctx, objectName))
	})
}
