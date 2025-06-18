package etcd

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/cago-frame/cago/configs"

	"github.com/cago-frame/cago/configs/file"
	"github.com/cago-frame/cago/configs/source"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func init() {
	configs.RegistrySource("etcd", func(cfg *configs.Config, serialization file.Serialization) (source.Source, error) {
		etcdConfig := &Config{}
		if err := cfg.Scan(context.Background(), "etcd", etcdConfig); err != nil {
			return nil, err
		}
		var err error
		etcdConfig.Prefix = path.Join(etcdConfig.Prefix, string(cfg.Env), cfg.AppName)
		s, err := NewSource(etcdConfig, serialization)
		if err != nil {
			return nil, err
		}
		return s, nil
	})
}

type Config struct {
	Endpoints []string
	Username  string
	Password  string
	Prefix    string
}

type etcd struct {
	*clientv3.Client
	prefix        string
	serialization file.Serialization
}

func NewSource(cfg *Config, serialization file.Serialization) (source.Source, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:            cfg.Endpoints,
		Username:             cfg.Username,
		Password:             cfg.Password,
		DialTimeout:          10 * time.Second,
		DialKeepAliveTimeout: 10 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &etcd{
		Client:        cli,
		prefix:        cfg.Prefix,
		serialization: serialization,
	}, nil
}

func (e *etcd) Scan(ctx context.Context, key string, value interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := e.Client.Get(ctx, path.Join(e.prefix, key))
	if err != nil {
		return err
	}
	if len(resp.Kvs) == 0 {
		b, err := e.serialization.Marshal(value)
		if err != nil {
			return err
		}
		if _, err := e.Client.Put(ctx, path.Join(e.prefix, key), string(b)); err != nil {
			return err
		}
		return fmt.Errorf("etcd %w: %s", source.ErrNotFound, key)
	}
	return e.serialization.Unmarshal(resp.Kvs[0].Value, value)
}

func (e *etcd) Has(ctx context.Context, key string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := e.Client.Get(ctx, path.Join(e.prefix, key))
	if err != nil {
		return false, err
	}
	return len(resp.Kvs) > 0, nil
}

func (e *etcd) Watch(ctx context.Context, key string, callback func(event source.Event)) error {
	go func() {
		w := e.Client.Watch(ctx, path.Join(e.prefix, key))
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
