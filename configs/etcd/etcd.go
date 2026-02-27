package etcd

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/configs/file"
	"github.com/cago-frame/cago/configs/source"
	dbetcd "github.com/cago-frame/cago/database/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func init() {
	configs.RegistrySource("etcd", func(cfg *configs.Config, serialization file.Serialization) (source.Source, error) {
		etcdConfig := &Config{}
		if err := cfg.Scan(context.Background(), "etcd", etcdConfig); err != nil {
			return nil, err
		}
		etcdConfig.Prefix = path.Join(etcdConfig.Prefix, string(cfg.Env), cfg.AppName)
		s, err := NewSource(etcdConfig, serialization)
		if err != nil {
			return nil, err
		}
		return s, nil
	})
}

type Config struct {
	dbetcd.Config `mapstructure:",squash"`
	Prefix        string
}

type etcd struct {
	client        *dbetcd.Client
	prefix        string
	serialization file.Serialization
}

func NewSource(cfg *Config, serialization file.Serialization) (source.Source, error) {
	cli, err := dbetcd.NewClient(&cfg.Config)
	if err != nil {
		return nil, err
	}
	dbetcd.SetDefault(cli)
	return &etcd{
		client:        dbetcd.NewCacheClient(cli, dbetcd.WithCache()),
		prefix:        cfg.Prefix,
		serialization: serialization,
	}, nil
}

func (e *etcd) Scan(ctx context.Context, key string, value interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	fullKey := path.Join(e.prefix, key)
	resp, err := e.client.Get(ctx, fullKey)
	if err != nil {
		return fmt.Errorf("etcd %s: %w", key, err)
	}
	if len(resp.Kvs) == 0 {
		b, err := e.serialization.Marshal(value)
		if err != nil {
			return err
		}
		if _, err := e.client.Put(ctx, fullKey, string(b)); err != nil {
			return err
		}
		return fmt.Errorf("etcd %w: %s, initialized with default value", source.ErrNotFound, key)
	}
	if err := e.serialization.Unmarshal(resp.Kvs[0].Value, value); err != nil {
		return fmt.Errorf("etcd unmarshal %s: %w", key, err)
	}
	return nil
}

func (e *etcd) Has(ctx context.Context, key string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := e.client.Get(ctx, path.Join(e.prefix, key))
	if err != nil {
		return false, fmt.Errorf("etcd %s: %w", key, err)
	}
	return len(resp.Kvs) > 0, nil
}

func (e *etcd) Watch(ctx context.Context, key string, callback func(event source.Event)) error {
	go func() {
		w := e.client.Watch(ctx, path.Join(e.prefix, key))
		for v := range w {
			if v.Err() != nil {
				break
			}
			for _, ev := range v.Events {
				switch ev.Type {
				case clientv3.EventTypePut:
					callback(source.Update)
				case clientv3.EventTypeDelete:
					callback(source.Delete)
				}
			}
		}
	}()
	return nil
}
