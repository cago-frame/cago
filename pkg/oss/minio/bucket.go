package minio

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/cago-frame/cago/pkg/oss/oss"

	"github.com/minio/minio-go/v7"
)

type Bucket struct {
	client *Client
	bucket string
}

func (b *Bucket) PutObject(ctx context.Context, objectName string, r io.Reader, objectSize int64) error {
	_, err := b.client.client.PutObject(ctx, b.bucket, objectName, r,
		objectSize, minio.PutObjectOptions{})
	return err
}

func (b *Bucket) PreSignedPutObject(ctx context.Context, objectName string, expires time.Duration) (u *url.URL, err error) {
	return b.client.client.PresignedPutObject(ctx, b.bucket, objectName, expires)
}

func (b *Bucket) GetObject(ctx context.Context, objectName string) (oss.Object, error) {
	obj, err := b.client.client.GetObject(ctx, b.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return newObject(obj), nil
}

func (b *Bucket) PreSignedGetObject(ctx context.Context, objectName string, expires time.Duration) (u *url.URL, err error) {
	return b.client.client.PresignedGetObject(ctx, b.bucket, objectName, expires, url.Values{})
}

func (b *Bucket) RemoveObject(ctx context.Context, objectName string) error {
	return b.client.client.RemoveObject(ctx, b.bucket, objectName, minio.RemoveObjectOptions{})
}

func (b *Bucket) FileURL(ctx context.Context, objectName string) (string, error) {
	return b.client.url + "/" + b.bucket + "/" + objectName, nil
}
