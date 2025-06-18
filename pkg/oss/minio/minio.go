package minio

import (
	"context"

	"github.com/cago-frame/cago/pkg/oss/oss"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	URL             string
}

type Client struct {
	url    string
	client *minio.Client
	core   *minio.Core
}

func New(cfg *Config) (oss.Client, error) {
	// Initialize minio core object.
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Region: "us-east-1",
		Secure: false,
	}
	if cfg.UseSSL {
		opts.Secure = true
	}
	minioCore, err := minio.NewCore(cfg.Endpoint, &minio.Options{})
	if err != nil {
		return nil, err
	}
	minioClient, err := minio.New(cfg.Endpoint, opts)
	if err != nil {
		return nil, err
	}
	return &Client{client: minioClient, core: minioCore, url: cfg.URL}, nil
}

func (c *Client) ListBuckets(ctx context.Context) ([]*oss.BucketInfo, error) {
	list, err := c.core.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	ret := make([]*oss.BucketInfo, 0, len(list))
	for _, v := range list {
		ret = append(ret, &oss.BucketInfo{Name: v.Name})
	}
	return ret, nil
}

func (c *Client) Bucket(ctx context.Context, bucket string) (oss.Bucket, error) {
	return &Bucket{
		client: c,
		bucket: bucket,
	}, nil
}

func (c *Client) URL() string {
	return c.url
}
