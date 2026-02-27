package etcd

import (
	"context"
	"time"

	"github.com/cago-frame/cago/configs"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var defaultClient *clientv3.Client

type Config struct {
	Endpoints []string
	Username  string
	Password  string //nolint:gosec // G117
}

func Etcd(ctx context.Context, config *configs.Config) error {
	cfg := &Config{}
	if err := config.Scan(ctx, "etcd", cfg); err != nil {
		return err
	}
	client, err := NewClient(cfg)
	if err != nil {
		return err
	}
	defaultClient = client
	return nil
}

func NewClient(cfg *Config) (*clientv3.Client, error) {
	return clientv3.New(clientv3.Config{
		Endpoints:            cfg.Endpoints,
		Username:             cfg.Username,
		Password:             cfg.Password,
		DialTimeout:          10 * time.Second,
		DialKeepAliveTimeout: 10 * time.Second,
	})
}

func SetDefault(client *clientv3.Client) {
	defaultClient = client
}

func Default() *clientv3.Client {
	return defaultClient
}
