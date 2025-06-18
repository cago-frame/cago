package file

import (
	"context"
	"fmt"
	"os"

	"github.com/cago-frame/cago/configs/source"
)

type fileSource struct {
	path          string
	config        map[string]interface{}
	serialization Serialization
}

func NewSource(filename string, serialization Serialization) (source.Source, error) {
	f := &fileSource{path: filename, serialization: serialization, config: make(map[string]interface{})}
	b, err := f.Read()
	if err != nil {
		return nil, err
	}
	if err := f.serialization.Unmarshal(b, &f.config); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *fileSource) Read() ([]byte, error) {
	return os.ReadFile(f.path)
}

func (f *fileSource) Scan(ctx context.Context, key string, value interface{}) error {
	cfg, ok := f.config[key]
	if !ok {
		f.config[key] = value
		b, err := f.serialization.Marshal(f.config)
		if err != nil {
			return err
		}
		if err := os.WriteFile(f.path, b, 0644); err != nil {
			return err
		}
		return fmt.Errorf("file %w: %s", source.ErrNotFound, key)
	}
	var b, err = f.serialization.Marshal(cfg)
	if err != nil {
		return err
	}
	return f.serialization.Unmarshal(b, value)
}

func (f *fileSource) Has(ctx context.Context, key string) (bool, error) {
	_, ok := f.config[key]
	return ok, nil
}

func (f *fileSource) Watch(ctx context.Context, key string, callback func(event source.Event)) error {
	return nil
}
